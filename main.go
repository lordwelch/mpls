package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

const (
	ChapterSearchMask = 1 << iota
	TimeSearchMask
	SkipToNextPointMask
	SkipBackToPreviousPointMask
	ForwardPlayMask
	BackwardPlayMask
	PlayMask
	StopMask
	PauseOnMask
	PauseOffMask
	StillOffMask
	ResumeMask
	MoveUpSelectedButtonMask
	MoveDownSelectedButtonMask
	MoveLeftSelectedButtonMask
	MoveRightSelectedButtonMask
	SelectButtonMask
	ActivateAndActivateMask
	SelectAndActivateMask
	AudioChangeMask
	PgTextstChangeMask
	AngleChangeMask
	PopupOnMask
	PopupOffMask
	SelectMenuLanguageMask
)

const (
	PlaylistRandomAccess = 1 << iota
	AudioMixApp
	LosslessMayBypassMixer
	// reserved
)

type MPLS struct {
	Header             string
	playlistStart      int
	playlistMarkStart  int
	extensionDataStart int
	AppInfoPlaylist    AppInfoPlaylist
	Playlist           Playlist
}

type AppInfoPlaylist struct {
	Len                  int
	PlaybackType         int
	PlaybackCount        int
	UOMask               uint64
	AppInfoPlaylistFlags uint16
}
type Playlist struct {
	len               int
	NumberOfPlayItems uint16
	numberOfSubpaths  uint16
	PlayItems         PlayItem
}

// reserved = 1 << (iota + 7)
const (
	IsDifferentAudios = 1 << (iota + 7)
	IsSeamlessAngleChange
)

type PlayItem struct {
	len uint16

	ClipFile string
	ClipID   string // M2TS

	// Reserved 11 bits
	IsMultiAngle        bool // (1 bit)
	ConnectionCondition byte // (4 bits)

	STCID   byte
	InTime  uint16
	OutTime uint16

	UOMask uint64

	RandomAccessFlag byte // 1 bit - 7 reserved

	StillMode byte

	stillTime  uint16
	angleCount byte
	AngleFlag  byte
}

type CLPI struct {
	ClipFile string
	ClipID   string // M2TS
	STCID    byte
}

func main() {
	parse(os.Args[1])
}
func parse(filename string) error {
	var (
		buf  [10]byte
		n    int
		n64  int64
		Mpls MPLS
	)
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	file := bytes.NewReader(f)

	n, err = file.Read(buf[:8])
	if err != nil || n != 8 {
		return err
	}
	str := string(buf[:8])
	if str[:4] != "MPLS" {
		return fmt.Errorf("%s is not an mpls file it must start with 'MPLS' it started with '%s'", filename, str[:4])
	}
	if str[4:8] != "0200" {
		fmt.Fprintf(os.Stderr, "warning: mpls may not work it is version %s\n", str[4:8])
	}

	Mpls.Header = str

	Mpls.playlistStart, err = readInt32(file, buf[:4])
	if err != nil {
		return err
	}
	fmt.Println("uint:", Mpls.playlistStart, "binary:", buf[:4])
	Mpls.playlistMarkStart, err = readInt32(file, buf[:4])
	if err != nil {
		return err
	}
	fmt.Println("uint:", Mpls.playlistMarkStart, "binary:", buf[:4])
	Mpls.extensionDataStart, err = readInt32(file, buf[:4])
	if err != nil {
		return err
	}
	fmt.Println("uint:", Mpls.extensionDataStart, "binary:", buf[:4])
	n64, err = file.Seek(20, io.SeekCurrent)
	if err != nil || n64 != 20 {
		return err
	}
	Mpls.AppInfoPlaylist.Len, err = readInt32(file, buf[:4])
	if err != nil {
		return err
	}
	fmt.Println("uint:", Mpls.AppInfoPlaylist.Len, "binary:", buf[:4])

	n, err = file.Read(buf[:4])
	if err != nil || n != 1 {
		return err
	}
	Mpls.AppInfoPlaylist.PlaybackType = int(buf[1])
	switch Mpls.AppInfoPlaylist.PlaybackType {
	case 2, 3:
		Mpls.AppInfoPlaylist.PlaybackCount = int(binary.BigEndian.Uint16(buf[3:4]))
		fmt.Println("uint:", Mpls.AppInfoPlaylist.PlaybackCount, "binary:", buf[3:4])
	}
	Mpls.AppInfoPlaylist.UOMask, err = readUInt64(file, buf[:8])
	if err != nil || n != 1 {
		return err
	}
	Mpls.AppInfoPlaylist.AppInfoPlaylistFlags, err = readUInt16(file, buf[:2])
	if err != nil || n != 1 {
		return err
	}
	err = Mpls.Playlist.parsePlaylist(file, int64(Mpls.playlistStart))
	if err != nil {
		return err
	}
	return nil
}

func (p Playlist) parsePlaylist(file io.ReadSeeker, PlaylistStart int64) error {
	var (
		n64 int64
		err error
		buf [10]byte
	)
	n64, err = file.Seek(PlaylistStart, io.SeekStart)
	if err != nil || n64 != 20 {
		return err
	}
	fmt.Println("uint:", PlaylistStart, "binary:", buf[:4])
	p.len, err = readInt32(file, buf[:4])
	if err != nil {
		return err
	}

	file.Read(buf[:5])
	return nil
}

func readUInt16(file io.Reader, buf []byte) (uint16, error) {
	n, err := file.Read(buf[:2])
	if err != nil || n != 2 {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf[:2]), nil
}

func readInt32(file io.Reader, buf []byte) (int, error) {
	n, err := file.Read(buf[:4])
	if err != nil || n != 4 {
		return 0, err
	}
	return int(binary.BigEndian.Uint32(buf[:4])), nil
}

func readUInt64(file io.Reader, buf []byte) (uint64, error) {
	n, err := file.Read(buf[:8])
	if err != nil || n != 8 {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf[:8]), nil
}
