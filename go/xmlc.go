package main

import (
    "bytes"
    "compress/gzip"
    "crypto/aes"
    "crypto/cipher"
    "crypto/md5"
    "encoding/binary"
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "time"
)

type Input string

func (in *Input) String() string {
    return string(*in)
}

func (in *Input) Set(value string) error {
    const invalidError = "Input file is invalid"

    if len(value) == 0 {
        return errors.New(invalidError)
    }

    stat, err := os.Stat(value)

    if os.IsNotExist(err) {
        return errors.New("Input file does not exist")
    }

    if stat.IsDir() || stat.Size() <= 0 {
        return errors.New(invalidError)
    }

    *in = Input(value)

    return nil
}

type Output string

func (out *Output) String() string {
    return string(*out)
}

func (out *Output) Set(value string) error {
    const invalidError = "Output file is invalid"

    if len(value) == 0 {
        return errors.New(invalidError)
    }

    stat, err := os.Stat(value)

    if os.IsExist(err) && stat.IsDir() {
        return errors.New(invalidError)
    }

    *out = Output(value)

    return nil
}

type Header struct {
    Magic        [4]byte
    Epoch        uint32
    Date         int32
    Sku          uint16
    Unknown      uint16
    SerialNumber [16]byte
}

func (h Header) isValid() bool {
    return bytes.Compare(h.Magic[:], []byte(xmlc)) == 0 && h.Epoch == uint32(y2k)
}

func (h Header) GenerateKeys(aes []byte, iv []byte) error {
    var hash [16]byte

    buf := new(bytes.Buffer)

    binary.Write(buf, binary.LittleEndian, uint32(pepper1))
    binary.Write(buf, binary.LittleEndian, h)

    hash = md5.Sum(buf.Bytes())

    copy(aes[:], hash[:])

    buf.Reset()

    binary.Write(buf, binary.LittleEndian, uint32(pepper2))

    hash = md5.Sum(buf.Bytes())

    copy(aes[16:], hash[:])

    buf.Reset()

    binary.Write(buf, binary.LittleEndian, uint32(pepper3))

    hash = md5.Sum(buf.Bytes())

    copy(iv[:], hash[:])

    return nil
}

func makeHeader() Header {
    h := Header{}

    copy(h.Magic[:], []byte(xmlc))

    h.Epoch = uint32(y2k)

    start := time.Date(2000, 3, 1, 0, 0, 0, 0, time.UTC)
    end := time.Now()

    h.Date = int32(end.Sub(start).Seconds())

    return h
}

type Context struct {
    compress bool
    in       Input
    out      Output
}

func (ctx Context) CompressAndEncode() error {
    in, err := os.Open(ctx.in.String())

    if err != nil {
        return err
    }

    defer in.Close()

    var buf bytes.Buffer

    zw := gzip.NewWriter(&buf)

    if _, err := io.Copy(zw, in); err != nil {
        return io.ErrUnexpectedEOF
    }

    if err := zw.Close(); err != nil {
        return err
    }

    hdr := makeHeader()

    key := make([]byte, 32)
    iv := make([]byte, 16)

    hdr.GenerateKeys(key, iv)

    blk, _ := aes.NewCipher(key)

    ofb := cipher.NewOFB(blk, iv[:])

    rdr := &cipher.StreamReader{S: ofb, R: &buf}

    out, err := os.Create(ctx.out.String())

    if err != nil {
        return err
    }

    defer out.Close()

    binary.Write(out, binary.LittleEndian, hdr)

    if _, err := io.Copy(out, rdr); err != nil {
        return err
    }

    return nil
}

func (ctx Context) DecodeAndExpand() error {
    in, err := os.Open(ctx.in.String())

    if err != nil {
        return err
    }

    defer in.Close()

    hdr := Header{}

    if err := binary.Read(in, binary.LittleEndian, &hdr); err != nil {
        return err
    }

    if !hdr.isValid() {
       return errors.New("Invalid XMLC")
    }

    key := make([]byte, 32)
    iv := make([]byte, 16)

    hdr.GenerateKeys(key, iv)

    blk, _ := aes.NewCipher(key)

    ofb := cipher.NewOFB(blk, iv[:])

    rdr := &cipher.StreamReader{S: ofb, R: in}

    var buf bytes.Buffer

    if _, err := io.Copy(&buf, rdr); err != nil {
        return err
    }

    zr, err := gzip.NewReader(&buf)

    if err != nil {
        return err
    }

    defer zr.Close()

    zr.Multistream(false)

    out, err := os.Create(ctx.out.String())

    if err != nil {
        return err
    }

    defer out.Close()

    if _, err := io.Copy(out, zr); err != nil {
        return err
    }

    return nil
}

const (
    empty = ""
    pepper1 = 0x78563412
    pepper2 = 0x75d7bc0a
    pepper3 = 0xb7542a91
    xmlc = "XMLC"
    y2k = 0x20000301
)

var (
    version string
    ctx     Context
)

func init() {
    flag.BoolVar(&ctx.compress, "c", false, empty)
    flag.Var(&ctx.in, "in", empty)
    flag.Var(&ctx.out, "out", empty)

    flag.Usage = func() {
        fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-c] <in> <out>\n", flag.CommandLine.Name())
    }
}

func main() {
    ver := flag.Bool("V", false, empty)

    flag.Parse()

    for !flag.Parsed() {
    }

    if *ver {
        fmt.Fprintf(flag.CommandLine.Output(), "%s ver. %s\n", flag.CommandLine.Name(), version)
        os.Exit(0)
    }

    if len(ctx.in) == 0 || len(ctx.out) == 0 {
        if flag.NArg() < 2 {
            flag.Usage()
            os.Exit(1)
        }

        if err := ctx.in.Set(flag.Arg(0)); err != nil {
            log.Fatal(err)
        }

        if err := ctx.out.Set(flag.Arg(1)); err != nil {
            log.Fatal(err)
        }
    }

    if ctx.compress {
        if err := ctx.CompressAndEncode(); err != nil {
            log.Fatal(err)
        }
    } else {
        if err := ctx.DecodeAndExpand(); err != nil {
            log.Fatal(err)
        }
    }
}
