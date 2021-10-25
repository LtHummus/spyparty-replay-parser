package spyparty

const VersionOffset = 0x06

type offsets struct {
	version int

	uuid              int
	duration          int
	timestamp         int
	sequenceNumber    int
	spyNameLength     int
	SniperNameLength  int
	gameResult        int
	venueHashOffset   int
	missionsCompleted int
	playerNames       int
	loadout           int

	spyDisplayNameLength    int
	sniperDisplaynameLength int
	startDuration           int
	numGuests               int
}

var version4Offsets = offsets{
	version: 4,

	uuid:     0x14,
	duration: 0x18,
}

var version5Offsets = offsets{
	version: 5,

	uuid: 0x14,
}

var version6Offsets = offsets{
	version: 6,

	uuid:              0x18,
	duration:          0x14,
	timestamp:         0x28,
	sequenceNumber:    0x2c,
	spyNameLength:     0x2e,
	SniperNameLength:  0x2f,
	gameResult:        0x38,
	venueHashOffset:   0x40,
	missionsCompleted: 0x50,
	playerNames:       0x64,
	loadout:           0x3c,

	spyDisplayNameLength:    0x30,
	sniperDisplaynameLength: 0x31,
	startDuration:           0x58,
	numGuests:               0x54,
}
