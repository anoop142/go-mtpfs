package mtp

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hanwen/usb"
)

func candidateFromDeviceDescriptor(d *usb.Device) *Device {
	dd, err := d.GetDeviceDescriptor()
	if err != nil {
		return nil
	}
	for i := byte(0); i < dd.NumConfigurations; i++ {
		cdecs, err := d.GetConfigDescriptor(i)
		if err != nil {
			return nil
		}
		for _, iface := range cdecs.Interfaces {
			for _, a := range iface.AltSetting {
				if len(a.EndPoints) != 3 {
					continue
				}
				m := Device{}
				for _, s := range a.EndPoints {
					switch {
					case s.Direction() == usb.ENDPOINT_IN && s.TransferType() == usb.TRANSFER_TYPE_INTERRUPT:
						m.eventEP = s.EndpointAddress
					case s.Direction() == usb.ENDPOINT_IN && s.TransferType() == usb.TRANSFER_TYPE_BULK:
						m.fetchEP = s.EndpointAddress
					case s.Direction() == usb.ENDPOINT_OUT && s.TransferType() == usb.TRANSFER_TYPE_BULK:
						m.sendEP = s.EndpointAddress
					}
				}
				if m.sendEP > 0 && m.fetchEP > 0 && m.eventEP > 0 {
					m.devDescr = *dd
					m.ifaceDescr = a
					m.dev = d.Ref()
					m.configValue = cdecs.ConfigurationValue
					return &m
				}
			}
		}
	}

	return nil
}

// FindDevices finds likely MTP devices without opening them.
func FindDevices(c *usb.Context) ([]*Device, error) {
	l, err := c.GetDeviceList()
	if err != nil {
		return nil, err
	}

	var cands []*Device
	for _, d := range l {
		cand := candidateFromDeviceDescriptor(d)
		if cand != nil {
			cands = append(cands, cand)
		}
	}
	l.Done()

	return cands, nil
}

// selectDevice finds a device that matches given pattern
func selectDevice(cands []*Device, pattern string) (*Device, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	var found []*Device
	for _, cand := range cands {
		if err := cand.Open(); err != nil {
			continue
		}

		found = append(found, cand)
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no MTP devices found")
	}

	cands = found
	found = nil
	var ids []string
	deviceSelection := 0
	for i, cand := range cands {
		id, err := cand.ID()
		if err != nil {
			cand.Close()
			return nil, fmt.Errorf("Id dev %d: %v", i, err)
		}

		if pattern == "" || re.FindString(id) != "" {
			found = append(found, cand)
			ids = append(ids, id)
		} else {
			cand.Close()
			cand.Done()
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no device matched")
	}

	if len(found) > 1 {
		fmt.Println("More than 1 device found!")
		for i, id := range ids {
			fmt.Printf("%d. %s\n", i, id)
		}
		fmt.Printf("Enter the device no : ")
		fmt.Scanf("%v\n", &deviceSelection)
	}
	cand := found[deviceSelection]
	config, err := cand.h.GetConfiguration()
	if err != nil {
		return nil, fmt.Errorf("could not get configuration of %v: %v",
			ids[deviceSelection], err)
	}
	if config != cand.configValue {

		if err := cand.h.SetConfiguration(cand.configValue); err != nil {
			return nil, fmt.Errorf("could not set configuration of %v: %v",
				ids[deviceSelection], err)
		}
	}
	return found[deviceSelection], nil
}

// SelectDevice returns opened MTP device that matches the given pattern.
func SelectDevice(pattern string) (*Device, error) {
	c := usb.NewContext()

	devs, err := FindDevices(c)
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, fmt.Errorf("no MTP devices found")
	}

	return selectDevice(devs, pattern)
}

// printDevices prints id
func printDevices(cands []*Device) {
	var found []*Device

	for _, cand := range cands {
		if err := cand.Open(); err != nil {
			continue
		}

		found = append(found, cand)
	}

	if len(found) == 0 {
		log.Fatalf("no MTP devices found")
	}

	for _, cand := range found {
		id, err := cand.ID()
		if err != nil {
			/*  ignore error */
			cand.Close()
			continue
		}

		fmt.Println(id)

	}

}

// ListDevices list id of all mtp devices found
func ListDevices() {
	c := usb.NewContext()

	devs, err := FindDevices(c)
	if err != nil {
		log.Fatalln(err)
	}
	if len(devs) == 0 {
		log.Fatalln("no MTP devices found")
	}

	printDevices(devs)

}
