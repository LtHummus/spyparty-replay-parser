package spyparty

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

const BytesToRead = 512 // we only need the first 512 bytes

type Result byte

var (
	MissionWin   = 0
	SpyTimeout   = 1
	SpyShot      = 2
	CivilianShot = 3
	InProgress   = 4
)

var (
	ErrFile              = errors.New("could not read file")
	ErrNotSpyPartyReplay = errors.New("not a SpyParty replay")
	ErrUnknownVersion    = errors.New("unknown SpyParty replay file version")
	ErrParseError        = errors.New("unable to parse replay data")
)

type Replay struct {
	Spy               string
	Sniper            string
	StartTime         time.Time
	Result            Result
	Loadout           string
	Venue             string
	SequenceNumber    int
	UUID              string
	Version           int
	StartDuration     int
	NumGuests         int
	MissionsCompleted int
}

func readBytes(filename string) ([]byte, error) {
	stream, err := os.Open(filename)
	if err != nil {
		log.Warn().Err(err).Str("filename", filename).Msg("could not open file")
		return nil, err
	}
	defer stream.Close()

	buf := make([]byte, BytesToRead)
	bytesRead, err := stream.Read(buf)
	if err != nil {
		log.Warn().Err(err).Str("filename", filename).Msg("could not read file")
		return nil, err
	}

	if bytesRead != BytesToRead {
		log.Warn().Str("filename", filename).Msg("could not read enough bytes")
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

func ParseReplayFile(filename string) (*Replay, error) {
	header, err := readBytes(filename)
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
	ret.Venue = fmt.Sprintf("%x", binary.LittleEndian.Uint32(header[offsets.venueHashOffset:offsets.venueHashOffset+4]))
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
