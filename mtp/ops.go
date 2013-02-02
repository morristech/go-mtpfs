package mtp

import (
	"bytes"
	"fmt"
	"io"
)

func (d *Device) OpenSession() error {
	if d.session != nil {
		return fmt.Errorf("session already open")
	}
	var req, rep Container
	req.Code = OC_OpenSession
	req.Param = []uint32{1} // session
	err := d.RPC(&req, &rep, nil, nil, 0)
	if err != nil {
		return err
	}

	// TODO - libmtp checks for invalid transaction, I/O err?
	d.session = &Session{
		tid: 1,
		sid: 1,
	}
	return nil
}

func (d *Device) CloseSession() error {
	var req, rep Container
	req.Code = OC_CloseSession
	return d.RPC(&req, &rep, nil, nil, 0)
}

func (d *Device) GetDeviceInfo(info *DeviceInfo) error {
	var req, rep Container

	req.Code = OC_GetDeviceInfo
	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, info)
	if err != nil {
		return err
	}
	return err
}

func (d *Device) GetStorageIDs(info *Uint32Array) error {
	var req, rep Container
	req.Code = OC_GetStorageIDs
	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, info)
	return err
}

func (d *Device) GetObjectPropDesc(objPropCode, objFormatCode uint16, info *ObjectPropDesc) error {
	var req, rep Container
	req.Code = OC_MTP_GetObjectPropDesc
	req.Param = []uint32{uint32(objPropCode), uint32(objFormatCode)}
	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}

	err = Decode(&buf, info)
	return err
}

func (d *Device) GetObjectPropValue(objHandle uint32, objPropCode uint16, value interface{}) error {
	var req, rep Container
	var buf bytes.Buffer

	req.Code = OC_MTP_GetObjectPropValue
	req.Param = []uint32{objHandle, uint32(objPropCode)}
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, value)
	return err
}

func (d *Device) SetObjectPropValue(objHandle uint32, objPropCode uint16, value interface{}) error {
	var req, rep Container
	req.Code = OC_MTP_SetObjectPropValue
	req.Param = []uint32{objHandle, uint32(objPropCode)}
	var buf bytes.Buffer
	err := Encode(&buf, value)
	if err != nil {
		return err
	}

	return d.RPC(&req, &rep, nil, &buf, int64(buf.Len()))
}

func (d *Device) GetObjectPropsSupported(objFormatCode uint16, props *Uint16Array) error {
	var req, rep Container
	var buf bytes.Buffer

	req.Code = OC_MTP_GetObjectPropsSupported
	req.Param = []uint32{uint32(objFormatCode)}
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, props)
	return err
}

func (d *Device) GetDevicePropDesc(propCode uint16, info *DevicePropDesc) error {
	var req, rep Container
	req.Code = OC_GetDevicePropDesc
	req.Param = append(req.Param, uint32(propCode))

	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}

	err = info.Decode(&buf)
	return err
}

func (d *Device) SetDevicePropValue(propCode uint32, src interface{}) error {
	var req, rep Container
	req.Code = OC_SetDevicePropValue
	req.Param = []uint32{propCode}

	var buf bytes.Buffer
	err := Encode(&buf, src)
	if err != nil {
		return err
	}
	return d.RPC(&req, &rep, nil, &buf, int64(buf.Len()))
}

func (d *Device) GetDevicePropValue(propCode uint32, dest interface{}) error {
	var req, rep Container
	req.Code = OC_GetDevicePropValue
	req.Param = []uint32{propCode}

	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, dest)
	return err
}

func (d *Device) ResetDeviceProp(propCode uint32) error {
	_, err := d.GenericRPC(OC_ResetDevicePropValue, []uint32{propCode})
	return err
}

func (d *Device) Claim() error {
	if d.h == nil {
		return fmt.Errorf("device not open")
	}

	err := d.h.ClaimInterface(d.ifaceDescr.InterfaceNumber)
	if err == nil {
		d.claimed = true
	}

	return err
}

func (d *Device) GetStorageInfo(ID uint32, info *StorageInfo) error {
	var req, rep Container
	req.Code = OC_GetStorageInfo
	req.Param = []uint32{ID}
	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, info)
	if err != nil {
		return err
	}
	return err
}

// An operation that just involves standard containers, such as GetNumObjects
func (d *Device) GenericRPC(opcode uint16, params []uint32) ([]uint32, error) {
	var req, rep Container
	req.Code = opcode
	req.Param = params
	err := d.RPC(&req, &rep, nil, nil, 0)
	return rep.Param, err
}

func (d *Device) GetObjectHandles(storageID,
	objFormatCode, parent uint32,
	info *ObjectHandles) error {
	var req, rep Container
	req.Code = OC_GetObjectHandles
	req.Param = []uint32{storageID, objFormatCode, parent}
	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, info)
	return err
}

func (d *Device) GetObjectInfo(handle uint32, info *ObjectInfo) error {
	var req, rep Container
	req.Code = OC_GetObjectInfo
	req.Param = []uint32{handle}
	var buf bytes.Buffer
	err := d.RPC(&req, &rep, &buf, nil, 0)
	if err != nil {
		return err
	}
	err = Decode(&buf, info)
	if err != nil {
		return err
	}
	return err
}

func (d *Device) DeleteObject(handle uint32) error {
	_, err := d.GenericRPC(OC_DeleteObject, []uint32{handle, 0x0})
	return err
}

func (d *Device) SendObjectInfo(wantStorageID, wantParent uint32, info *ObjectInfo) (storageID, parent, handle uint32, err error) {
	var req, rep Container
	req.Code = OC_SendObjectInfo
	req.Param = []uint32{wantStorageID, wantParent}

	buf := &bytes.Buffer{}
	err = Encode(buf, info)
	if err != nil {
		return
	}

	err = d.RPC(&req, &rep, nil, buf, int64(buf.Len()))
	if err != nil {
		return
	}

	return rep.Param[0], rep.Param[1], rep.Param[2], nil
}

func (d *Device) SendObject(r io.Reader, size int64) error {
	var req, rep Container
	req.Code = OC_SendObject
	return d.RPC(&req, &rep, nil, r, size)
}

func (d *Device) GetObject(handle uint32, w io.Writer) error {
	var req, rep Container
	req.Code = OC_GetObject
	req.Param = []uint32{handle}

	return d.RPC(&req, &rep, w, nil, 0)
}

func (d *Device) GetPartialObject(handle uint32, w io.Writer, offset uint32, size uint32) error {
	var req, rep Container
	req.Code = OC_ANDROID_GET_PARTIAL_OBJECT64
	req.Param = []uint32{handle, offset, size}
	return d.RPC(&req, &rep, w, nil, 0)
}
