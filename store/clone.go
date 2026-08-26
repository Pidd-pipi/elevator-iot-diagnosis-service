package store

// Clone 返回困人观测记录的深拷贝，避免调用方修改仓储内部对象。
func (o *EntrapmentObservation) Clone() *EntrapmentObservation {
	if o == nil {
		return nil
	}
	c := *o
	if o.ReportIDs != nil {
		c.ReportIDs = append([]string(nil), o.ReportIDs...)
	}
	return &c
}
