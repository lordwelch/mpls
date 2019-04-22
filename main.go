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
	FileType           string
	Version            string
	PlaylistStart      int
	PlaylistMarkStart  int
	ExtensionDataStart int
	AppInfoPlaylist    AppInfoPlaylist
	Playlist           Playlist
}

// AppInfoPlaylist sucks
type AppInfoPlaylist struct {
	Len           int
	PlaybackType  byte
	PlaybackCount uint16
	PlaylistFlags uint16
	UOMask        uint64
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
	InTime           int
	OutTime          int
	UOMask           uint64
	RandomAccessFlag byte
	AngleCount       byte
	AngleFlags       byte
	StillMode        byte
	StillTime        uint16
	Clpi             CLPI
	Angles           []CLPI
	StreamTable      STNTable
}

// STNTable STream Number Table
type STNTable struct {
	Len                       uint16 // Reserved uint16
	PrimaryVideoStreamCount   byte
	PrimaryAudioStreamCount   byte
	PrimaryPGStreamCount      byte
	PrimaryIGStreamCount      byte
	SecondaryVideoStreamCount byte
	SecondaryAudioStreamCount byte
	PIPPGStreamCount          byte
	PrimaryVideoStreams       []PrimaryStream
	PrimaryAudioStreams       []PrimaryStream
	PrimaryPGStreams          []PrimaryStream
	PrimaryIGStreams          []PrimaryStream
	SecondaryAudioStreams     []SecondaryAudioStream
	SecondaryVideoStreams     []SecondaryVideoStream
}

// PrimaryStream holds a stream entry and attributes
type PrimaryStream struct {
	StreamEntry
	StreamAttributes
}

// SecondaryStream holds stream references
type SecondaryStream struct {
	RefrenceEntryCount byte
	StreamIDs          []byte
}

// SecondaryAudioStream holds a primary stream and a secondary stream
type SecondaryAudioStream struct {
	PrimaryStream
	ExtraAttributes SecondaryStream
}

// SecondaryVideoStream holds a primary stream and a secondary stream for the video
// and a secondary stream for the Presentation Graphics/pip
type SecondaryVideoStream struct {
	PrimaryStream
	ExtraAttributes SecondaryStream
	PGStream        SecondaryStream
}

// StreamEntry holds the information for the data stream
type StreamEntry struct {
	Len       byte
	Type      byte
	PID       uint16
	SubPathID byte
	SubClipID byte
}

// StreamAttributes holds metadata about the data stream
type StreamAttributes struct {
	Len           byte
	Encoding      byte
	Format        byte
	Rate          byte
	CharacterCode byte
	Language      string
}

// CLPI contains the fiLename and the codec ID
type CLPI struct {
	ClipFile string
	ClipID   string // M2TS
	STCID    byte
}

type SubPath struct {
	Len           uint32
	Type          byte
	PlayItemCount byte
	Flags         uint16
}

// SubPlayItem contains information about a PlayItem in the subpath
type SubPlayItem struct {
	Len              uint16
	Flags            byte // multiangle/connection condition
	Type             byte
	InTime           int
	OutTime          int
	StartOfPlayitem  uint64
	UOMask           uint64
	RandomAccessFlag byte
	AngleCount       byte
	AngleFlags       byte
	StillMode        byte
	StillTime        uint16
	PlayItemID       uint16
	Clpi             CLPI
	Angles           []CLPI
	StreamTable      STNTable
}

type errReader struct {
	RS  *bytes.Reader
	err error
}

func (er *errReader) Read(p []byte) (n int, err error) {
	if er.err != nil {
		return 0, er.err
	}

	n, er.err = er.RS.Read(p)
	if n != len(p) {
		er.err = fmt.Errorf("%s", "Invalid read")
	}

	return n, er.err
}

func (er *errReader) Seek(offset int64, whence int) (int64, error) {
	if er.err != nil {
		return 0, er.err
	}

	var n64 int64
	n64, er.err = er.Seek(offset, whence)

	return n64, er.err
}

func main() {
	var (
		file io.Reader
		Mpls MPLS
		err  error
	)
	file, err = os.Open(filepath.Clean(os.Args[1]))
	if err != nil {
		panic(err)
	}
	Mpls, err = Parse(file)
	fmt.Println(Mpls)
	panic(err)
}

// Parse parses an MPLS file into an MPLS struct
func Parse(reader io.Reader) (mpls MPLS, err error) {
	var (
		file []byte
	)

	file, err = ioutil.ReadAll(reader)
	if err != nil {
		return MPLS{}, err
	}

	err = mpls.Parse(file)
	return mpls, err
}

// Parse reads MPLS data from an io.ReadSeeker
func (mpls *MPLS) Parse(file []byte) error {
	var (
		buf [10]byte
		n   int
		err error
	)

	reader := &errReader{
		RS:  bytes.NewReader(file),
		err: nil,
	}
	n, err = reader.Read(buf[:8])
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

	mpls.FileType = str[:4]
	mpls.Version = str[4:8]

	mpls.PlaylistStart, _ = readInt32(reader, buf[:])
	fmt.Println("int:", mpls.PlaylistStart, "binary:", buf[:4])

	mpls.PlaylistMarkStart, _ = readInt32(reader, buf[:])
	fmt.Println("int:", mpls.PlaylistMarkStart, "binary:", buf[:4])

	mpls.ExtensionDataStart, _ = readInt32(reader, buf[:])
	fmt.Println("int:", mpls.ExtensionDataStart, "binary:", buf[:4])

	_, _ = reader.Seek(20, io.SeekCurrent)

	_ = mpls.AppInfoPlaylist.parse(reader)

	_, _ = reader.Seek(int64(mpls.PlaylistStart), io.SeekStart)

	_ = mpls.Playlist.parse(reader)

	return reader.err
}

// parse reads AppInfoPlaylist data from an *errReader
func (aip *AppInfoPlaylist) parse(reader *errReader) error {
	var (
		buf [10]byte
	)
	aip.Len, _ = readInt32(reader, buf[:])
	fmt.Println("int:", aip.Len, "binary:", buf[:4])

	_, _ = reader.Read(buf[:1])

	aip.PlaybackType = buf[1]
	fmt.Println("int:", aip.PlaybackType, "binary:", buf[1])

	aip.PlaybackCount, _ = readUInt16(reader, buf[:])
	fmt.Println("int:", aip.PlaybackCount, "binary:", buf[:2])

	aip.UOMask, _ = readUInt64(reader, buf[:])
	fmt.Println("int:", aip.UOMask, "binary:", buf[:8])

	aip.PlaylistFlags, _ = readUInt16(reader, buf[:])
	fmt.Println("int:", aip.PlaylistFlags, "binary:", buf[:2])

	return reader.err
}

// parse reads Playlist data from an *errReader
func (p *Playlist) parse(reader *errReader) error {
	var (
		buf [10]byte
		err error
	)

	p.Len, _ = readInt32(reader, buf[:])
	fmt.Println("int:", p.Len, "binary:", buf[:4])

	_, _ = reader.Seek(2, io.SeekCurrent)

	p.PlayItemCount, _ = readUInt16(reader, buf[:])
	fmt.Println("int:", p.PlayItemCount, "binary:", buf[:2])

	p.SubPathCount, _ = readUInt16(reader, buf[:])
	fmt.Println("int:", p.SubPathCount, "binary:", buf[:2])

	for i := 0; i < int(p.PlayItemCount); i++ {
		var item PlayItem
		err = item.parse(reader)
		if err != nil {
			return err
		}
		p.PlayItems = append(p.PlayItems, item)
	}

	for i := 0; i < int(p.SubPathCount); i++ {

	}

	return reader.err
}

// parse reads PlayItem data from an *errReader
func (pi *PlayItem) parse(reader *errReader) error {
	var (
		buf [10]byte
	)

	pi.Len, _ = readUInt16(reader, buf[:])
	fmt.Println("int:", pi.Len, "binary:", buf[:2])

	_, _ = reader.Read(buf[:9])

	str := string(buf[:9])
	if str[5:9] != "M2TS" {
		fmt.Fprintf(os.Stderr, "warning: this playlist may be faulty it has a play item that is '%s' not 'M2TS'", str[4:8])
	}
	pi.Clpi.ClipFile = str[:5]
	pi.Clpi.ClipID = str[5:9]

	pi.Flags, _ = readUInt16(reader, buf[:])

	_, _ = reader.Read(buf[:1])

	pi.Clpi.STCID = buf[0]

	pi.InTime, _ = readInt32(reader, buf[:])

	pi.OutTime, _ = readInt32(reader, buf[:])

	pi.UOMask, _ = readUInt64(reader, buf[:])

	_, _ = reader.Read(buf[:2])

	pi.RandomAccessFlag = buf[0]

	pi.StillMode = buf[1]

	pi.StillTime, _ = readUInt16(reader, buf[:])

	if pi.Flags&1 == 1 {
		_, _ = reader.Read(buf[:2])

		pi.AngleCount = buf[0]

		pi.AngleFlags = buf[1]

		for i := 0; i < int(pi.AngleCount); i++ {
			var angle CLPI
			_ = angle.parse(reader)

			pi.Angles = append(pi.Angles, angle)
		}
	}

	_ = pi.StreamTable.parse(reader)

	return reader.err
}

// parse reads angle data from an *errReader
func (clpi *CLPI) parse(reader *errReader) error {
	var (
		buf [10]byte
	)
	_, _ = reader.Read(buf[:])

	str := string(buf[:9])
	clpi.ClipFile = str[:5]
	clpi.ClipID = str[5:9]

	clpi.STCID = buf[9]
	return reader.err
}

// parse reads PrimaryStream data from an *errReader
func (stnt *STNTable) parse(reader *errReader) error {
	var (
		buf [10]byte
		err error
	)

	stnt.Len, _ = readUInt16(reader, buf[:])

	_, _ = reader.Read(buf[:9])

	stnt.PrimaryVideoStreamCount = buf[2]
	stnt.PrimaryAudioStreamCount = buf[3]
	stnt.PrimaryPGStreamCount = buf[4]
	stnt.PrimaryIGStreamCount = buf[5]
	stnt.SecondaryAudioStreamCount = buf[6]
	stnt.SecondaryVideoStreamCount = buf[7]
	stnt.PIPPGStreamCount = buf[8]

	_, _ = reader.Seek(5, io.SeekCurrent)

	for i := 0; i < int(stnt.PrimaryVideoStreamCount); i++ {
		var stream PrimaryStream
		err = stream.parse(reader)
		if err != nil {
			return err
		}
		stnt.PrimaryVideoStreams = append(stnt.PrimaryVideoStreams, stream)
	}

	for i := 0; i < int(stnt.PrimaryAudioStreamCount); i++ {
		var stream PrimaryStream
		err = stream.parse(reader)
		if err != nil {
			return err
		}
		stnt.PrimaryAudioStreams = append(stnt.PrimaryAudioStreams, stream)
	}

	for i := 0; i < int(stnt.PrimaryPGStreamCount); i++ {
		var stream PrimaryStream
		err = stream.parse(reader)
		if err != nil {
			return err
		}
		stnt.PrimaryPGStreams = append(stnt.PrimaryPGStreams, stream)
	}

	for i := 0; i < int(stnt.PrimaryIGStreamCount); i++ {
		var stream PrimaryStream
		err = stream.parse(reader)
		if err != nil {
			return err
		}
		stnt.PrimaryIGStreams = append(stnt.PrimaryIGStreams, stream)
	}

	for i := 0; i < int(stnt.SecondaryAudioStreamCount); i++ {
		var stream SecondaryAudioStream
		err = stream.parse(reader)
		if err != nil {
			return err
		}
		stnt.SecondaryAudioStreams = append(stnt.SecondaryAudioStreams, stream)
	}

	for i := 0; i < int(stnt.SecondaryVideoStreamCount); i++ {
		var stream SecondaryVideoStream
		err = stream.parse(reader)
		if err != nil {
			return err
		}
		stnt.SecondaryVideoStreams = append(stnt.SecondaryVideoStreams, stream)
	}

	return reader.err
}

// parse reads SecondaryStream data from an *errReader
func (ss *SecondaryStream) parse(reader *errReader) error {
	var (
		buf [10]byte
	)

	_, _ = reader.Read(buf[:2])
	ss.RefrenceEntryCount = buf[0]
	ss.StreamIDs = make([]byte, ss.RefrenceEntryCount)
	_, _ = reader.Read(ss.StreamIDs)
	if ss.RefrenceEntryCount%2 != 0 {
		_, _ = reader.Seek(1, io.SeekCurrent)
	}
	return reader.err
}

// parse reads SecondaryAudioStream data from an *errReader
func (sas *SecondaryAudioStream) parse(reader *errReader) error {
	_ = sas.PrimaryStream.parse(reader)
	_ = sas.ExtraAttributes.parse(reader)

	return reader.err
}

// parse reads SecondaryVideoStream data from an *errReader
func (svs *SecondaryVideoStream) parse(reader *errReader) error {
	_ = svs.PrimaryStream.parse(reader)
	_ = svs.ExtraAttributes.parse(reader)
	_ = svs.PGStream.parse(reader)

	return reader.err
}

// parse reads Stream data from an *errReader
func (ps *PrimaryStream) parse(reader *errReader) error {

	_ = ps.StreamEntry.parse(reader)

	_ = ps.StreamAttributes.parse(reader)

	return reader.err
}

// parse reads Stream data from an *errReader
func (se *StreamEntry) parse(reader *errReader) error {
	var (
		buf [10]byte
	)

	_, _ = reader.Read(buf[:])

	se.Len = buf[0]
	se.Type = buf[1]
	switch se.Type {
	case 1:
		se.PID = binary.BigEndian.Uint16(buf[2:4])
	case 2, 4:
		se.SubPathID = buf[2]
		se.SubClipID = buf[3]
		se.PID = binary.BigEndian.Uint16(buf[4:6])
	case 3:
		se.SubPathID = buf[2]
		se.PID = binary.BigEndian.Uint16(buf[3:5])
	}

	return reader.err
}

// parse reads Stream data from an *errReader
func (sa *StreamAttributes) parse(reader *errReader) error {
	var (
		buf [10]byte
	)
	_, _ = reader.Read(buf[:2])

	sa.Len = buf[0]
	sa.Encoding = buf[1]

	switch sa.Encoding {
	case VTMPEG1Video, VTMPEG2Video, VTVC1, VTH264:
		_, _ = reader.Read(buf[:1])

		sa.Format = buf[0] & 0xf0 >> 4
		sa.Rate = buf[0] & 0x0F

	case ATMPEG1Audio, ATMPEG2Audio, ATLPCM, ATAC3, ATDTS, ATTRUEHD, ATAC3Plus, ATDTSHD, ATDTSHDMaster:
		_, _ = reader.Read(buf[:4])

		sa.Format = buf[0] & 0xf0 >> 4
		sa.Rate = buf[0] & 0x0F
		sa.Language = string(buf[1:4])

	case PresentationGraphics, InteractiveGraphics:
		_, _ = reader.Read(buf[:3])

		sa.Language = string(buf[:3])

	case TextSubtitle:
		_, _ = reader.Read(buf[:4])

		sa.CharacterCode = buf[0]
		sa.Language = string(buf[1:4])
	default:
		fmt.Fprintf(os.Stderr, "warning: unrecognized encoding: '%02X'", sa.Encoding)
	}

	return reader.err
}

func readUInt16(reader io.Reader, buf []byte) (uint16, error) {
	n, err := reader.Read(buf[:2])
	if err != nil || n != 2 {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf[:2]), nil
}

func readInt32(reader io.Reader, buf []byte) (int, error) {
	n, err := readUInt32(reader, buf)
	return int(n), err
}

func readUInt32(reader io.Reader, buf []byte) (uint32, error) {
	n, err := reader.Read(buf[:4])
	if err != nil || n != 4 {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf[:4]), nil
}

func readUInt64(reader io.Reader, buf []byte) (uint64, error) {
	n, err := reader.Read(buf[:8])
	if err != nil || n != 8 {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf[:8]), nil
}
