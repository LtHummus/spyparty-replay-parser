package spyparty

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"time"
)

const BytesToRead = 512 // we only need the first 512 bytes

type Result byte

var (
	MissionWin   = Result(0)
	SpyTimeout   = Result(1)
	SpyShot      = Result(2)
	CivilianShot = Result(3)
	InProgress   = Result(4)
)

var (
	Spy    = "spy"
	Sniper = "sniper"
)

var (
	ErrFile              = errors.New("could not read file")
	ErrNotSpyPartyReplay = errors.New("not a SpyParty replay")
	ErrUnknownVersion    = errors.New("unknown SpyParty replay file version")
	ErrParseError        = errors.New("unable to parse replay data")
	ErrUnknownVenue      = errors.New("unknown venue hash")
)

type Replay struct {
	Spy               string    `json:"spy"`
	Sniper            string    `json:"sniper"`
	StartTime         time.Time `json:"start_time"`
	Result            Result    `json:"result"`
	Loadout           string    `json:"loadout"`
	Venue             string    `json:"venue"`
	SequenceNumber    int       `json:"sequence_number"`
	UUID              string    `json:"uuid"`
	Version           int       `json:"version"`
	StartDuration     int       `json:"start_duration,omitempty"`
	NumGuests         int       `json:"num_guests,omitempty"`
	MissionsCompleted int       `json:"missions_completed"`
}

func readBytes(source io.Reader) ([]byte, error) {
	buf := make([]byte, BytesToRead)
	bytesRead, err := source.Read(buf)
	if err != nil {
		return nil, err
	}

	if bytesRead != BytesToRead {
		return nil, err
	}

	return buf, nil
}

func getOffsets(version byte) *offsets {
	switch version {
	case 4:
		return &version4Offsets
	case 6:
		return &version6Offsets
	default:
		return nil
	}
}

func parseLoadout(encodedLoadout uint32) string {
	modeNum := encodedLoadout >> 28
	y := (encodedLoadout & 0x0FFFC000) >> 14
	x := encodedLoadout & 0x00003FFF

	switch modeNum {
	case 0:
		return fmt.Sprintf("k%d", x)
	case 1:
		return fmt.Sprintf("p%d/%d", x, y)
	case 2:
		return fmt.Sprintf("a%d/%d", x, y)
	default:
		return ""
	}
}

func SupportedVersions() []int {
	return []int{6}
}

func ParseReplayFile(source io.Reader) (*Replay, error) {
	header, err := readBytes(source)
	if err != nil {
		return nil, ErrFile
	}

	// check magic number
	if header[0] != 'R' || header[1] != 'P' || header[2] != 'L' || header[3] != 'Y' {
		return nil, ErrNotSpyPartyReplay
	}

	version := header[4]

	offsets := getOffsets(version)
	if offsets == nil {
		return nil, ErrUnknownVersion
	}

	ret := &Replay{}

	ret.Result = Result(header[offsets.gameResult])

	unixTime := binary.LittleEndian.Uint32(header[offsets.timestamp : offsets.timestamp+4])
	ret.StartTime = time.Unix(int64(unixTime), 0)

	loadout := parseLoadout(binary.LittleEndian.Uint32(header[offsets.loadout : offsets.loadout+4]))
	if loadout == "" {
		return nil, ErrParseError
	}
	ret.Loadout = loadout
	venueHash := binary.LittleEndian.Uint32(header[offsets.venueHashOffset : offsets.venueHashOffset+4])
	venue := Venues[venueHash]
	if venue == "" {
		return ret, ErrUnknownVenue
	} else {
		ret.Venue = venue
	}
	ret.UUID = base64.RawURLEncoding.EncodeToString(header[offsets.uuid : offsets.uuid+16])
	ret.SequenceNumber = int(binary.LittleEndian.Uint16(header[offsets.sequenceNumber : offsets.sequenceNumber+2]))

	if offsets.numGuests != 0 {
		ret.NumGuests = int(binary.LittleEndian.Uint32(header[offsets.numGuests : offsets.numGuests+4]))
	}

	if offsets.startDuration != 0 {
		ret.StartDuration = int(binary.LittleEndian.Uint32(header[offsets.startDuration : offsets.startDuration+4]))
	}

	spyNameLength := int(header[offsets.spyNameLength])
	sniperNameLength := int(header[offsets.SniperNameLength])
	spyDisplayNameLength := int(header[offsets.spyDisplayNameLength])
	sniperDisplayNameLength := int(header[offsets.sniperDisplaynameLength])

	if spyDisplayNameLength != 0 {
		trueOffset := offsets.playerNames + spyNameLength + sniperNameLength
		ret.Spy = string(header[trueOffset : trueOffset+spyDisplayNameLength])
	} else {
		ret.Spy = string(header[offsets.playerNames : offsets.playerNames+spyNameLength])
	}

	if sniperDisplayNameLength != 0 {
		trueOffset := offsets.playerNames + spyNameLength + sniperNameLength + spyDisplayNameLength
		ret.Sniper = string(header[trueOffset : trueOffset+sniperDisplayNameLength])
	} else {
		trueOffset := offsets.playerNames + spyNameLength
		ret.Sniper = string(header[trueOffset : trueOffset+sniperNameLength])
	}

	ret.MissionsCompleted = bits.OnesCount32(binary.LittleEndian.Uint32(header[offsets.missionsCompleted : offsets.missionsCompleted+4]))

	return ret, nil
}

func (r *Replay) WinnerRole() string {
	if r.Result == MissionWin || r.Result == CivilianShot {
		return Spy
	} else if r.Result == SpyShot || r.Result == SpyTimeout {
		return Sniper
	} else {
		return ""
	}
}

func (r *Replay) WinnerName() string {
	if r.WinnerRole() == Spy {
		return r.Spy
	} else if r.WinnerRole() == Sniper {
		return r.Sniper
	} else {
		return ""
	}
}
