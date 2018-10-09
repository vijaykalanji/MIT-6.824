package mapreduce

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTask int, // which reduce task this is
	outFile string, // write the output here
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {
	//
	// doReduce manages one reduce task: it should read the intermediate
	// files for the task, sort the intermediate key/value pairs by key,
	// call the user-defined reduce function (reduceF) for each key, and
	// write reduceF's output to disk.
	//
	// You'll need to read one intermediate file from each map task;
	// reduceName(jobName, m, reduceTask) yields the file
	// name from map task m.
	//
	// Your doMap() encoded the key/value pairs in the intermediate
	// files, so you will need to decode them. If you used JSON, you can
	// read and decode by creating a decoder and repeatedly calling
	// .Decode(&kv) on it until it returns an error.
	//
	// You may find the first example in the golang sort package
	// documentation useful.
	//
	// reduceF() is the application's reduce function. You should
	// call it once per distinct key, with a slice of all the values
	// for that key. reduceF() returns the reduced value for that key.
	//
	// You should write the reduce output as JSON encoded KeyValue
	// objects to the file named outFile. We require you to use JSON
	// because that is what the merger than combines the output
	// from all the reduce tasks expects. There is nothing special about
	// JSON -- it is just the marshalling format we chose to use. Your
	// output code will look something like this:
	//
	// enc := json.NewEncoder(file)
	// for key := ... {
	// 	enc.Encode(KeyValue{key, reduceF(...)})
	// }
	// file.Close()
	//
	// Your code here (Part I).
	//
	fmt.Println(jobName);
	fmt.Println(reduceTask);
	fmt.Println(outFile);
	fmt.Println(nMap);

	// Each map tak will create one file for every reducer.
	// So, we need to loop over all the map results for this

	hMap := make(map[string][]string)
	for i:=0 ; i<nMap;i++ {
		//Get the file name
		mapFileName:=reduceName(jobName, i, reduceTask)
		fh,err:= os.Open(mapFileName)
		//fileHandle, err := os.OpenFile(mapFileName, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
		panic(err)
		}
		dec := json.NewDecoder(fh)
		fmt.Println("From REDUCE")
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			//fmt.Printf("%s: %s\n", kv.Key, kv.Value)
			hMap[kv.Key] = append(hMap[kv.Key],kv.Value)

		}
		fh.Close()
		//Values are in Hash Map.
		//fHandle,error1:= os.Open(outFile)
		//fHandle,error1:= os.OpenFile(outFile, os.O_RDONLY|os.O_CREATE, 0666)
		fHandle, error1 := os.OpenFile(outFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if error1 != nil {
			panic(err)
		}
		//fmt.Println("outFile-->",outFile)
		enc := json.NewEncoder(fHandle)
		fmt.Println("Getting ready to write to a file")
		for k, v := range hMap {
			//fmt.Printf("key-->Value")
			//fmt.Printf("key[%s] value[%s]\n", k, v)
			//kv1 := KeyValue{k,  reduceF(k,v)}
			//fmt.Println("--->",kv1)
			enc.Encode(KeyValue{k,  reduceF(k,v)})
		}
		fHandle.Close()
	}


}
