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
	UOChapterSearchMask = 1 << iota
	UOTimeSearchMask
	UOSkipToNextPointMask
	UOSkipBackToPreviousPointMask
	UOForwardPlayMask
	UOBackwardPlayMask
	UOPlayMask
	UOStopMask
	UOPauseOnMask
	UOPauseOffMask
	UOStillOffMask
	UOResumeMask
	UOMoveUpSelectedButtonMask
	UOMoveDownSelectedButtonMask
	UOMoveLeftSelectedButtonMask
	UOMoveRightSelectedButtonMask
	UOSelectButtonMask
	UOActivateAndActivateMask
	UOSelectAndActivateMask
	UOAudioChangeMask
	UOPgTextstChangeMask
	UOAngleChangeMask
	UOPopupOnMask
	UOPopupOffMask
	UOSelectMenuLanguageMask
)

// Playlist Flags
const (
	PFPlaylistRandomAccess = 1 << iota
	PFAudioMixApp
	PFLosslessMayBypassMixer
	PFreserved
)

// Angle Flags
const (
	AFIsDifferentAudios = 1 << (iota + 7)
	AFIsSeamlessAngleChange
)

// VideoType
const (
	VTMPEG1Video = 0x01
	VTMPEG2Video = 0x02
	VTVC1        = 0xea
	VTH264       = 0x1b
)

// AudioType
const (
	ATMPEG1Audio  = 0x03
	ATMPEG2Audio  = 0x04
	ATLPCM        = 0x80
	ATAC3         = 0x81
	ATDTS         = 0x82
	ATTRUEHD      = 0x83
	ATAC3Plus     = 0x84
	ATDTSHD       = 0x85
	ATDTSHDMaster = 0x86
)

// OtherType
const (
	PresentationGraphics = 0x90
	InteractiveGraphics  = 0x91
	TextSubtitle         = 0x92
)

// VideoFormat
const (
	VFReserved = iota
	VF480I
	VF576I
	VF480P
	VF1080I
	VF720P
	VF1080P
	VF576P
)

// FrameRate
const (
	FRReserved = iota
	FR23976    // 23.976
	FR24       // 24
	FR25       // 25
	FR2997     // 29.97
	FR50       // 50
	FR5994     // 59.94
)

// AspectRatio
const (
	ARReserved = 0
	AR43       = 2 //4:3
	AR169      = 3 //16:9
)

// AudioPresentation
const (
	APReserved = 0
	APMono     = 1
	APDualMono = 2
	APStereo   = 3
	APMulti    = 6
	APCombo    = 12
)

// SampleRate
const (
	SRReserved = 0
	SR48       = 1
	SR96       = 4
	SR192      = 5
	SR48192    = 12 // 48/192
	SR4896     = 14 // 48/96
)

// CharacterCode
const (
	ReservedCharacterCode = iota
	UTF8
	UTF16
	ShiftJIS // Japanese
	KSC5601  // Korean
	GB18030  // Chinese
	GB2312   // Chinese
	BIG5     // Chinese
) // Chinese

// MPLS is a struct representing an MPLS file
type MPLS struct {
	Header             string
	PlaylistStart      int
	PlaylistMarkStart  int
	ExtensionDataStart int
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
	Len           int
	PlayItemCount uint16
	SubPathCount  uint16
	PlayItems     []PlayItem
}

// PlayItem contains information about a an item in the playlist
type PlayItem struct {
	Len              uint16
	Flags            uint16 // multiangle/connection condition
	StillTime        uint16
	Clpi             CLPI
	InTime           int
	OutTime          int
	UOMask           uint64
	StillMode        byte
	STCID            byte
	RandomAccessFlag byte
	AngleCount       byte
	AngleFlags       byte
	Angles           []CLPI
}

// CLPI contains the fiLename and the codec ID
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
func Parse(fiLename string) (Mpls MPLS, err error) {
	var (
		file *bytes.Reader
		f    []byte
	)

	f, err = ioutil.ReadFile(filepath.Clean(fiLename))
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

	Mpls.PlaylistStart, err = readInt32(file, buf[:4])
	fmt.Println("int:", Mpls.PlaylistStart, "binary:", buf[:4])
	if err != nil {
		return err
	}

	Mpls.PlaylistMarkStart, err = readInt32(file, buf[:4])
	fmt.Println("int:", Mpls.PlaylistMarkStart, "binary:", buf[:4])
	if err != nil {
		return err
	}

	Mpls.ExtensionDataStart, err = readInt32(file, buf[:4])
	fmt.Println("int:", Mpls.ExtensionDataStart, "binary:", buf[:4])
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

	_, err = file.Seek(int64(Mpls.PlaylistStart), io.SeekStart)
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

	p.Len, err = readInt32(file, buf[:])
	fmt.Println("int:", p.Len, "binary:", buf[:4])
	if err != nil {
		return err
	}
	_, err = file.Seek(2, io.SeekCurrent)
	if err != nil {
		return err
	}
	p.PlayItemCount, err = readUInt16(file, buf[:])
	fmt.Println("int:", p.PlayItemCount, "binary:", buf[:2])
	if err != nil {
		return err
	}
	p.SubPathCount, err = readUInt16(file, buf[:])
	fmt.Println("int:", p.SubPathCount, "binary:", buf[:2])
	if err != nil {
		return err
	}
	for i := 0; i < int(p.PlayItemCount); i++ {
		var item PlayItem
		err = item.Parse(file)
		if err != nil {
			return err
		}
		p.PlayItems = append(p.PlayItems, item)
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

	pi.Len, err = readUInt16(file, buf[:])
	fmt.Println("int:", pi.Len, "binary:", buf[:2])
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
	pi.Clpi.ClipFile = str[:5]
	pi.Clpi.ClipID = str[5:9]

	pi.Flags, err = readUInt16(file, buf[:])
	if err != nil {
		return err
	}

	n, err = file.Read(buf[:1])
	if err != nil || n != 1 {
		return err
	}
	pi.STCID = buf[0]

	pi.InTime, err = readInt32(file, buf[:])
	if err != nil {
		return err
	}

	pi.OutTime, err = readInt32(file, buf[:])
	if err != nil {
		return err
	}

	pi.UOMask, err = readUInt64(file, buf[:])
	if err != nil {
		return err
	}

	n, err = file.Read(buf[:1])
	if err != nil || n != 1 {
		return err
	}
	pi.RandomAccessFlag = buf[0]

	n, err = file.Read(buf[:1])
	if err != nil || n != 1 {
		return err
	}
	pi.StillMode = buf[0]

	pi.StillTime, err = readUInt16(file, buf[:])
	if err != nil {
		return err
	}

	if pi.Flags&1 == 1 {
		n, err = file.Read(buf[:1])
		if err != nil || n != 1 {
			return err
		}
		pi.AngleCount = buf[0]

		n, err = file.Read(buf[:1])
		if err != nil || n != 1 {
			return err
		}
		pi.AngleFlags = buf[0]

		for i := 0; i < int(pi.AngleCount); i++ {
			var angle CLPI
			err = angle.Parse(file)
			if err != nil {
				return err
			}
			pi.Angles = append(pi.Angles, angle)
		}
	}

	return nil
}

// Parse reads angle data from an io.ReadSeeker
func (clpi *CLPI) Parse(file io.Reader) error {
	var (
		buf [10]byte
		n   int
		err error
	)
	n, err = file.Read(buf[:9])
	if err != nil || n != 9 {
		return err
	}
	str := string(buf[:9])
	clpi.ClipFile = str[:5]
	clpi.ClipID = str[5:9]

	n, err = file.Read(buf[:1])
	if err != nil || n != 1 {
		return err
	}
	clpi.STCID = buf[0]
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
