package common

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentMap(t *testing.T) {
	currentMap := NewCurrentMap()
	wg := sync.WaitGroup{}
	count := 100
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				currentMap.Set(1, 1)
				val, ok := currentMap.Get(1)
				assert.True(t, ok)
				assert.EqualValues(t, 1, val)
				assert.IsType(t, 1, val)
				//fmt.Printf("val %v,type:%T\n", val, val)

				currentMap.Set("1", 2)
				val, ok = currentMap.Get("1")
				assert.True(t, ok)
				assert.EqualValues(t, 2, val)
				assert.IsType(t, 1, val)
				//fmt.Printf("val %v,type:%T\n", val, val)

				val, ok = currentMap.Get(1)
				assert.True(t, ok)
				assert.EqualValues(t, 1, val)
				assert.IsType(t, 1, val)

				currentMap.Remove(3)

				//fmt.Printf("val %v,type:%T\n", val, val)
				assert.EqualValues(t, 2, currentMap.Count(), "")
				for range currentMap.IterBuffered() {
					//fmt.Println(v.Key, "=", v.Val)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	mapItems := currentMap.Items()
	assert.EqualValues(t, 2, len(mapItems))
	for k, v := range mapItems {
		fmt.Println(k, "=", v)
	}
	fmt.Println("finish")
}
