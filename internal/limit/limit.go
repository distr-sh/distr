package limit

type Limit int64

const Unlimited Limit = -1

func New(v int64) Limit {
	if v < 0 {
		return Unlimited
	} else {
		return Limit(v)
	}
}

func (l Limit) Value() int64 {
	return int64(l)
}

func (l Limit) IsUnlimited() bool {
	return l == Unlimited
}

func (l Limit) IsReached(other int64) bool {
	return !l.IsUnlimited() && int64(l) <= other
}

func (l Limit) IsExceeded(other int64) bool {
	return !l.IsUnlimited() && int64(l) < other
}
