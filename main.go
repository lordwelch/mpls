package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// User Operation mask table
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

// Playlist Flags
const (
	PlaylistRandomAccess = 1 << iota
	AudioMixApp
	LosslessMayBypassMixer
	reserved
)

const (
	IsDifferentAudios = 1 << (iota + 7)
	IsSeamlessAngleChange
)

// MPLS is a struct representing an MPLS file
type MPLS struct {
	Header             string
	playlistStart      int
	playlistMarkStart  int
	extensionDataStart int
	AppInfoPlaylist    AppInfoPlaylist
	Playlist           Playlist
}

// AppInfoPlaylist sucks
type AppInfoPlaylist struct {
	Len           int
	PlaybackType  int
	PlaybackCount int
	UOMask        uint64
	PlaylistFlags uint16
}

// Playlist sucks
type Playlist struct {
	len           int
	playItemCount uint16
	subPathCount  uint16
	playItems     []PlayItem
}

// PlayItem contains information about a an item in the playlist
type PlayItem struct {
	len              uint16
	clpi             CLPI
	flags            uint16 // multiangle/connection condition
	inTime           int32
	outTime          int32
	UOMask           uint64
	RandomAccessFlag byte
	stillMode        byte
	stillTime        uint16
	angleCount       byte
	angleFlags       byte
	angles           []CLPI
}

// CLPI contains the filename and the codec ID
type CLPI struct {
	ClipFile string
	ClipID   string // M2TS
	STCID    byte
}

func main() {
	Mpls, err := Parse(os.Args[1])
	fmt.Println(Mpls)
	panic(err)
}

// Parse parses an MPLS file into an MPLS struct
func Parse(filename string) (Mpls MPLS, err error) {
	var (
		file *bytes.Reader
		f    []byte
	)

	f, err = ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		return MPLS{}, err
	}
	file = bytes.NewReader(f)
	err = Mpls.Parse(file)
	return Mpls, err
}

// Parse reads MPLS data from an io.ReadSeeker
func (Mpls *MPLS) Parse(file io.ReadSeeker) error {
	var (
		buf [10]byte
		n   int
		err error
	)

	n, err = file.Read(buf[:8])
	if err != nil || n != 8 {
		return err
	}
	str := string(buf[:8])
	if str[:4] != "MPLS" {
		return fmt.Errorf("not an mpls file it must start with 'MPLS' it started with '%s'", str[:4])
	}
	if str[4:8] != "0200" {
		fmt.Fprintf(os.Stderr, "warning: mpls may not work it is version %s\n", str[4:8])
	}

	Mpls.Header = str

	Mpls.playlistStart, err = readInt32(file, buf[:4])
	fmt.Println("int:", Mpls.playlistStart, "binary:", buf[:4])
	if err != nil {
		return err
	}

	Mpls.playlistMarkStart, err = readInt32(file, buf[:4])
	fmt.Println("int:", Mpls.playlistMarkStart, "binary:", buf[:4])
	if err != nil {
		return err
	}

	Mpls.extensionDataStart, err = readInt32(file, buf[:4])
	fmt.Println("int:", Mpls.extensionDataStart, "binary:", buf[:4])
	if err != nil {
		return err
	}

	_, err = file.Seek(20, io.SeekCurrent)
	if err != nil {
		return err
	}
	err = Mpls.AppInfoPlaylist.parse(file)
	if err != nil {
		return err
	}

	_, err = file.Seek(int64(Mpls.playlistStart), io.SeekStart)
	if err != nil {
		return err
	}

	err = Mpls.Playlist.Parse(file)
	if err != nil {
		return err
	}
	return nil
}

// Parse reads AppInfoPlaylist data from an io.ReadSeeker
func (aip *AppInfoPlaylist) parse(file io.ReadSeeker) error {
	var (
		buf [10]byte
		err error
		n   int
	)
	aip.Len, err = readInt32(file, buf[:4])
	fmt.Println("int:", aip.Len, "binary:", buf[:4])
	if err != nil {
		return err
	}

	n, err = file.Read(buf[:4])
	if err != nil || n != 4 {
		return err
	}
	aip.PlaybackType = int(buf[1])
	fmt.Println("int:", aip.PlaybackType, "binary:", buf[1])

	aip.PlaybackCount = int(binary.BigEndian.Uint16(buf[2:4]))
	fmt.Println("int:", aip.PlaybackCount, "binary:", buf[2:4])

	aip.UOMask, err = readUInt64(file, buf[:8])
	fmt.Println("int:", aip.UOMask, "binary:", buf[:8])
	if err != nil || n != 1 {
		return err
	}
	aip.PlaylistFlags, err = readUInt16(file, buf[:2])
	fmt.Println("int:", aip.PlaylistFlags, "binary:", buf[:2])
	if err != nil || n != 1 {
		return err
	}
	return nil
}

// Parse reads Playlist data from an io.ReadSeeker
func (p *Playlist) Parse(file io.ReadSeeker) error {
	var (
		buf [10]byte
		err error
	)

	p.len, err = readInt32(file, buf[:])
	fmt.Println("int:", p.len, "binary:", buf[:4])
	if err != nil {
		return err
	}
	_, err = file.Seek(2, io.SeekCurrent)
	if err != nil {
		return err
	}
	p.playItemCount, err = readUInt16(file, buf[:])
	fmt.Println("int:", p.playItemCount, "binary:", buf[:2])
	if err != nil {
		return err
	}
	p.subPathCount, err = readUInt16(file, buf[:])
	fmt.Println("int:", p.subPathCount, "binary:", buf[:2])
	if err != nil {
		return err
	}
	for i := 0; i < int(p.playItemCount); i++ {
		var item PlayItem
		err = item.Parse(file)
		if err != nil {
			return err
		}
		p.playItems = append(p.playItems, item)
	}

	return nil
}

// Parse reads PlayItem data from an io.ReadSeeker
func (pi *PlayItem) Parse(file io.Reader) error {
	var (
		buf [10]byte
		n   int
		err error
	)

	pi.len, err = readUInt16(file, buf[:])
	fmt.Println("int:", pi.len, "binary:", buf[:2])
	if err != nil {
		return err
	}
	n, err = file.Read(buf[:9])
	if err != nil || n != 9 {
		return err
	}
	str := string(buf[:9])
	if str[5:9] != "M2TS" {
		fmt.Fprintf(os.Stderr, "warning: this playlist may be faulty it has a play item that is '%s' not 'M2TS'", str[4:8])
	}
	pi.clpi.file = str[:5]
	pi.clpi.Codec = str[5:9]

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
	n, err := readUInt32(file, buf)
	return int(n), err
}

func readUInt32(file io.Reader, buf []byte) (uint32, error) {
	n, err := file.Read(buf[:4])
	if err != nil || n != 4 {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf[:4]), nil
}

func readUInt64(file io.Reader, buf []byte) (uint64, error) {
	n, err := file.Read(buf[:8])
	if err != nil || n != 8 {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf[:8]), nil
}
