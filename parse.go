package mpls

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

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
	n64, er.err = er.RS.Seek(offset, whence)

	return n64, er.err
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
		buf   [10]byte
		n     int
		err   error
		start int64
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

	mpls.PlaylistMarkStart, _ = readInt32(reader, buf[:])

	mpls.ExtensionDataStart, _ = readInt32(reader, buf[:])

	_, _ = reader.Seek(20, io.SeekCurrent)

	_ = mpls.AppInfoPlaylist.parse(reader)

	start, _ = reader.Seek(0, io.SeekCurrent)
	if start != int64(mpls.PlaylistStart) {
		fmt.Fprintf(os.Stderr, "Playlist doesn't start at the right place. Current position is %d position should be %d\n", start, int64(mpls.PlaylistStart))
	}

	_, _ = reader.Seek(int64(mpls.PlaylistStart), io.SeekStart)
	_ = mpls.Playlist.parse(reader)

	start, _ = reader.Seek(0, io.SeekCurrent)
	if start != int64(mpls.PlaylistMarkStart) {
		fmt.Fprintf(os.Stderr, "Mark Playlist doesn't start at the right place. Current position is %d position should be %d\n", start, int64(mpls.PlaylistStart))
	}

	// _ = mpls.MarkPlaylist.parse(reader)

	return reader.err
}

// parse reads AppInfoPlaylist data from an *errReader
func (aip *AppInfoPlaylist) parse(reader *errReader) error {
	var (
		buf   [10]byte
		start int64
		end   int64
	)

	aip.Len, _ = readInt32(reader, buf[:])

	start, _ = reader.Seek(0, io.SeekCurrent)

	_, _ = reader.Read(buf[:2])

	aip.PlaybackType = buf[1]

	aip.PlaybackCount, _ = readUInt16(reader, buf[:])

	aip.UOMask, _ = readUInt64(reader, buf[:])

	aip.PlaylistFlags, _ = readUInt16(reader, buf[:])

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(aip.Len)) {
		fmt.Fprintf(os.Stderr, "App Info Playlist is not aligned. App Info Playlist started at %d current position is %d position should be %d\n", start, end, start+int64(aip.Len))
	}

	return reader.err
}

// parse reads Playlist data from an *errReader
func (p *Playlist) parse(reader *errReader) error {
	var (
		buf   [10]byte
		err   error
		start int64
		end   int64
	)

	p.Len, _ = readInt32(reader, buf[:])

	start, _ = reader.Seek(0, io.SeekCurrent)

	_, _ = reader.Seek(2, io.SeekCurrent)

	p.PlayItemCount, _ = readUInt16(reader, buf[:])

	p.SubPathCount, _ = readUInt16(reader, buf[:])

	for i := 0; i < int(p.PlayItemCount); i++ {
		var item PlayItem
		err = item.parse(reader)
		if err != nil {
			return err
		}
		p.PlayItems = append(p.PlayItems, item)
	}

	for i := 0; i < int(p.SubPathCount); i++ {
		var item SubPath
		err = item.parse(reader)
		if err != nil {
			return err
		}
		p.SubPaths = append(p.SubPaths, item)
	}

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(p.Len)) {
		fmt.Fprintf(os.Stderr, "Playlist is not aligned. Playlist started at %d current position is %d position should be %d\n", start, end, start+int64(p.Len))
	}

	return reader.err
}

// parse reads PlayItem data from an *errReader
func (pi *PlayItem) parse(reader *errReader) error {
	var (
		buf   [10]byte
		err   error
		start int64
		end   int64
	)

	pi.Len, _ = readUInt16(reader, buf[:])

	start, _ = reader.Seek(0, io.SeekCurrent)

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

	if pi.Flags&1<<3 == 1 {
		_, _ = reader.Read(buf[:2])

		pi.AngleCount = buf[0]

		pi.AngleFlags = buf[1]

		for i := 0; i < int(pi.AngleCount); i++ {
			var angle CLPI
			_ = angle.parse(reader)
			_, err = reader.Read(buf[:1])
			if err != nil {
				return err
			}
			angle.STCID = buf[0]
			pi.Angles = append(pi.Angles, angle)
		}
	}

	_ = pi.StreamTable.parse(reader)

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(pi.Len)) {
		fmt.Fprintf(os.Stderr, "playitem is not aligned. Playitem started at %d current position is %d position should be %d\n", start, end, start+int64(pi.Len))
	}

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

	// clpi.STCID = buf[9]
	return reader.err
}

// parse reads PrimaryStream data from an *errReader
func (stnt *STNTable) parse(reader *errReader) error {
	var (
		buf   [10]byte
		err   error
		start int64
		end   int64
	)
	stnt.Len, _ = readUInt16(reader, buf[:])

	start, _ = reader.Seek(0, io.SeekCurrent)

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

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(stnt.Len)) {
		fmt.Fprintf(os.Stderr, "STN Table is not aligned. STN Table started at %d current position is %d position should be %d\n", start, end, start+int64(stnt.Len))
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
		buf   [10]byte
		start int64
		end   int64
	)

	_, _ = reader.Read(buf[:1])

	se.Len = buf[0]

	start, _ = reader.Seek(0, io.SeekCurrent)

	_, _ = reader.Read(buf[:9])
	se.Type = buf[0]
	switch se.Type {
	case 1:
		se.PID = binary.BigEndian.Uint16(buf[1:3])
	case 2, 4:
		se.SubPathID = buf[1]
		se.SubClipID = buf[2]
		se.PID = binary.BigEndian.Uint16(buf[3:5])
	case 3:
		se.SubPathID = buf[1]
		se.PID = binary.BigEndian.Uint16(buf[2:4])
	}

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(se.Len)) {
		fmt.Fprintf(os.Stderr, "Stream Entry is not aligned. Stream Entry started at %d current position is %d position should be %d\n", start, end, start+int64(se.Len))
	}

	return reader.err
}

// parse reads Stream data from an *errReader
func (sa *StreamAttributes) parse(reader *errReader) error {
	var (
		buf   [10]byte
		start int64
		end   int64
	)

	_, _ = reader.Read(buf[:1])

	sa.Len = buf[0]

	start, _ = reader.Seek(0, io.SeekCurrent)

	_, _ = reader.Read(buf[:1])

	sa.Encoding = buf[0]

	switch sa.Encoding {
	case VTMPEG1Video, VTMPEG2Video, VTVC1, VTH264:
		_, _ = reader.Read(buf[:1])

		sa.Format = buf[0] & 0xf0 >> 4
		sa.Rate = buf[0] & 0x0F
		_, _ = reader.Seek(3, io.SeekCurrent)

	case ATMPEG1Audio, ATMPEG2Audio, ATLPCM, ATAC3, ATDTS, ATTRUEHD, ATAC3Plus, ATDTSHD, ATDTSHDMaster:
		_, _ = reader.Read(buf[:4])

		sa.Format = buf[0] & 0xf0 >> 4
		sa.Rate = buf[0] & 0x0F
		sa.Language = string(buf[1:4])

	case PresentationGraphics, InteractiveGraphics:
		_, _ = reader.Read(buf[:3])

		sa.Language = string(buf[:3])
		_, _ = reader.Seek(1, io.SeekCurrent)

	case TextSubtitle:
		_, _ = reader.Read(buf[:4])

		sa.CharacterCode = buf[0]
		sa.Language = string(buf[1:4])
	default:
		fmt.Fprintf(os.Stderr, "warning: unrecognized encoding: '%02X'\n", sa.Encoding)
	}

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(sa.Len)) {
		fmt.Fprintf(os.Stderr, "Stream Attributes is not aligned. Stream Attributes started at %d current position is %d position should be %d\n", start, end, start+int64(sa.Len))
	}

	return reader.err
}

func (sp *SubPath) parse(reader *errReader) error {
	var (
		buf   [10]byte
		err   error
		start int64
		end   int64
	)

	sp.Len, _ = readInt32(reader, buf[:])

	start, _ = reader.Seek(0, io.SeekCurrent)

	_, _ = reader.Read(buf[:2])
	sp.Type = buf[1]
	sp.Flags, _ = readUInt16(reader, buf[:])

	_, _ = reader.Read(buf[:2])
	sp.PlayItemCount = buf[1]

	for i := 0; i < int(sp.PlayItemCount); i++ {
		var item SubPlayItem
		err = item.parse(reader)
		if err != nil {
			return err
		}
		sp.SubPlayItems = append(sp.SubPlayItems, item)
	}

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(sp.Len)) {
		fmt.Fprintf(os.Stderr, "Subpath is not aligned. Subpath started at %d current position is %d position should be %d\n", start, end, start+int64(sp.Len))
	}

	return reader.err
}

func (spi *SubPlayItem) parse(reader *errReader) error {
	var (
		buf   [10]byte
		err   error
		start int64
		end   int64
	)

	spi.Len, _ = readUInt16(reader, buf[:])

	start, _ = reader.Seek(0, io.SeekCurrent)

	_ = spi.Clpi.parse(reader)

	_, _ = reader.Read(buf[:4])

	spi.Flags = buf[2]
	spi.Clpi.STCID = buf[3]

	spi.InTime, _ = readInt32(reader, buf[:])
	spi.OutTime, _ = readInt32(reader, buf[:])

	spi.PlayItemID, _ = readUInt16(reader, buf[:])
	spi.StartOfPlayitem, _ = readUInt32(reader, buf[:])

	if spi.Flags&1<<3 == 1 {
		_, _ = reader.Read(buf[:2])

		spi.AngleCount = buf[0]
		spi.AngleFlags = buf[1]

		for i := 0; i < int(spi.AngleCount); i++ {
			var angle CLPI
			_ = angle.parse(reader)
			_, err = reader.Read(buf[:1])
			if err != nil {
				return err
			}
			angle.STCID = buf[0]
			spi.Angles = append(spi.Angles, angle)
		}
	}

	end, _ = reader.Seek(0, io.SeekCurrent)
	if end != (start + int64(spi.Len)) {
		fmt.Fprintf(os.Stderr, "Subplayitem is not aligned. Subplayitem started at %d current position is %d position should be %d\n", start, end, start+int64(spi.Len))
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
