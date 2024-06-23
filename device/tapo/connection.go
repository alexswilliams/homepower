package tapo

type tapoDeviceConnection interface {
	forgetKeysAndSession()
	GetDeviceInfo() (map[string]interface{}, error)
	GetEnergyUsage() (map[string]interface{}, error)
}

type lazyDeviceConnection struct {
	email    string
	password string
	deviceIp string
	port     uint16
	delegate tapoDeviceConnection
}

func (dc *lazyDeviceConnection) forgetKeysAndSession() {
	if dc.delegate != nil {
		dc.delegate.forgetKeysAndSession()
	}
}
func (dc *lazyDeviceConnection) GetDeviceInfo() (map[string]interface{}, error) {
	if dc.delegate == nil {
		err := dc.choose()
		if err != nil {
			return nil, err
		}
	}
	return dc.delegate.GetDeviceInfo()
}

func (dc *lazyDeviceConnection) GetEnergyUsage() (map[string]interface{}, error) {
	if dc.delegate == nil {
		err := dc.choose()
		if err != nil {
			return nil, err
		}
	}
	return dc.delegate.GetEnergyUsage()
}

func (dc *lazyDeviceConnection) choose() error {
	klap, err := newKlapDeviceConnection(dc.email, dc.password, dc.deviceIp, dc.port)
	if err != nil {
		return err
	}
	err = klap.doKeyExchange()
	if err == nil {
		dc.delegate = klap
		return err
	}

	oldTapo, err := newTapoOldDeviceConnection(dc.email, dc.password, dc.deviceIp, dc.port)
	if err != nil {
		return err
	}
	dc.delegate = oldTapo
	return nil
}

func connectionFactory(email, password, deviceIp string, port uint16) tapoDeviceConnection {
	return &lazyDeviceConnection{
		email:    email,
		password: password,
		deviceIp: deviceIp,
		port:     port,
		delegate: nil,
	}
}
