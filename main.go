package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"gopkg.in/mgo.v2/bson"

	mgo "gopkg.in/mgo.v2"
)

/*
 */
var readStdin = flag.Bool("i", false, "read file from stdin")
var offset = flag.Int("o", -1, "file offset of identifier in stdin")
var debug = flag.Bool("debug", false, "debug mode")
var tflag = flag.Bool("t", false, "print type information")
var aflag = flag.Bool("a", false, "print public type and member information")
var Aflag = flag.Bool("A", false, "print all type and members information")
var fflag = flag.String("f", "", "Go source filename")
var acmeFlag = flag.Bool("acme", false, "use current acme window")
var jsonFlag = flag.Bool("json", false, "output location in JSON format (-t flag is ignored)")
var renamegodef = flag.String("s", "godef", "in case you want to rename you godef,,,en or use the other tool instead of godef")

//mongo
var (
	URL        = "127.0.0.1:27017"
	mgoSession *mgo.Session
	dataBase   = "godefcache"
	collection = "godefcache"
)

func getSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial(URL)
		if err != nil {
			panic("dial wrong")
		}
	}
	return mgoSession.Clone()
}

// md5
var gFlagMD5 string

// pwd

func modifyOffset(src []byte) {
	for idx := *offset - 1; idx != -1; idx-- {
		if (src[idx] >= 'a' && src[idx] <= 'z') || (src[idx] >= 'A' && src[idx] <= 'z') || (src[idx] >= '0' && src[idx] <= '9') || src[idx] == '_' {
		} else {
			*offset = idx + 1
			break
		}
	}
}
func md55(input string, inputbytes []byte) string {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(input))
	md5Ctx.Write(inputbytes)
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}
func genFlagMD5() {
	flagStr := ""
	flagStr += fmt.Sprintf("%v\n", *readStdin)
	flagStr += fmt.Sprintf("%v\n", *offset)
	flagStr += fmt.Sprintf("%v\n", *debug)
	flagStr += fmt.Sprintf("%v\n", *tflag)

	flagStr += fmt.Sprintf("%v\n", *aflag)
	flagStr += fmt.Sprintf("%v\n", *Aflag)
	flagStr += fmt.Sprintf("%v\n", *fflag)
	flagStr += fmt.Sprintf("%v\n", *acmeFlag)
	flagStr += fmt.Sprintf("%v\n", *jsonFlag)
	gFlagMD5 = md55(flagStr, []byte{})
}

type godefcache struct {
	Raw string
}
type godefname struct {
	Name string
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: godefcache [flags] [expr]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	if *renamegodef != "godef" {
		getSession().DB(dataBase).C(collection).Upsert(bson.M{"_id": "toolname"}, bson.M{"$set": bson.M{"name": *renamegodef}})
		success(*renamegodef)
	}
	if !(*readStdin) {
		fail("%v", "Only support stdin now....")
	}
	src, _ := ioutil.ReadAll(os.Stdin)
	modifyOffset(src)
	genFlagMD5()
	md55(gFlagMD5, src)
	var result godefcache
	if err := getSession().DB(dataBase).C(collection).Find(bson.M{"_id": gFlagMD5}).One(&result); err != nil {
		var cmdstr string
		cmdstdin := bytes.NewBuffer(src)
		cmdstdout := bytes.NewBuffer([]byte{})
		cmdstderr := bytes.NewBuffer([]byte{})
		var godefnameresult godefname
		if err := getSession().DB(dataBase).C(collection).Find(bson.M{"_id": "toolname"}).One(&godefnameresult); err != nil {
			cmdstr = "godef"
		} else {
			cmdstr = godefnameresult.Name
		}
		godefcmd := exec.Command(cmdstr, "-i", "-t", "-o", fmt.Sprintf("%v", *offset), "-f", fmt.Sprintf("%v", *fflag))
		godefcmd.Stdin = cmdstdin
		godefcmd.Stdout = cmdstdout
		godefcmd.Stderr = cmdstderr
		if err := godefcmd.Run(); err == nil {
			raw := cmdstdout.String()
			getSession().DB(dataBase).C(collection).Insert(bson.M{"_id": gFlagMD5, "raw": raw})
			success(raw)
		} else {
			fail("%v", err.Error()+"[  --  ]"+string(cmdstderr.Bytes()))
		}
	}
	success(result.Raw)
}

///////////////////////////////////////////
func fail(s string, a ...interface{}) {
	fmt.Fprint(os.Stderr, "godef: "+fmt.Sprintf(s, a...)+"\n")
	os.Exit(2)
}
func success(s string) {
	fmt.Fprint(os.Stdout, s)
	os.Exit(0)
}