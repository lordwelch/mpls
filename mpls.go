package mpls

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
	MarkPlaylist       PlaylistMark
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
	SubPaths      []SubPath
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
	Len           int
	Type          byte
	PlayItemCount byte
	Flags         uint16
	SubPlayItems  []SubPlayItem
}

// SubPlayItem contains information about a PlayItem in the subpath
type SubPlayItem struct {
	Len              uint16
	Flags            byte // multiangle/connection condition
	StartOfPlayitem  uint32
	InTime           int
	OutTime          int
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

type PlaylistMark struct {
	Len       uint64
	MarkCount uint16
	Marks     []Mark
}

type Mark struct {
	Type        byte
	PlayItemRef uint16
	Time        uint32
	PID         uint16
	Duration    uint32
}
