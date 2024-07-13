package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
	hook "github.com/robotn/gohook"
)

var keyboard string = "default"
var ENTER uint16 = 36
var SPACE uint16 = 49

var sounds map[string][]byte = make(map[string][]byte)

var keyMap = make(map[uint16]bool)

func loadSoundsForKeyboard(keyboard string) {
	keys := []string{"down1", "up1", "down2", "up2", "down3", "up3", "down4", "up4", "down5", "up5", "down6", "up6", "down7", "up7", "down_space", "up_space", "down_enter", "up_enter", "up_mouse", "down_mouse"}
	for _, key := range keys {
		loadSound(keyboard, key)
	}
}

func loadSound(keyboard string, soundName string) {
	soundFile, err := os.Open(fmt.Sprintf("sounds/%s/%s.mp3", keyboard, soundName))

	if err != nil {
		log.Fatalf("failed to open sound file: %v", err)
	}
	defer soundFile.Close()

	sound, err := io.ReadAll(soundFile)
	if err != nil {
		log.Fatalf("failed to read sound file: %v", err)
	}

	sounds[soundName] = sound
}

func getRandomUpKey() string {
	keys := []string{"down1", "down2", "down3", "down4", "down5", "down6", "down7"}
	key := keys[rand.Intn(len(keys))]
	return key
}

func getRandomDownKey() string {
	keys := []string{"up1", "up2", "up3", "up4", "up5", "up6", "up7"}
	key := keys[rand.Intn(len(keys))]
	return key
}

func main() {

	if len(os.Args) > 1 {
		keyboard = os.Args[1]
	}

	if _, err := os.Stat(fmt.Sprintf("sounds/%s", keyboard)); os.IsNotExist(err) {
		log.Fatalf("Keyboard sounds not found: %s", keyboard)
	}

	fmt.Printf("Using keyboard: %s\n", keyboard)

	loadSoundsForKeyboard(keyboard)

	// Create an Oto context (for audio playback)
	context, err := oto.NewContext(48000, 2, 2, 8192)
	if err != nil {
		log.Fatalf("failed to create Oto context: %v", err)
	}
	defer context.Close()

	// Function to play sound in a goroutine
	playSound := func(key string) {
		// Create an MP3 decoder
		decoder, err := mp3.NewDecoder(bytes.NewReader(sounds[key]))
		if err != nil {
			log.Fatalf("failed to create MP3 decoder: %v", err)
		}

		player := context.NewPlayer()
		defer player.Close()

		// Reset the decoder (so it plays from the beginning)
		decoder.Seek(0, 0)

		// Create a buffer to read the decoded audio
		buf := make([]byte, 8192)
		for {
			n, err := decoder.Read(buf)
			if err != nil && err != io.EOF {
				log.Printf("failed to read decoded audio: %v", err)
				break
			}
			if n == 0 {
				break
			}
			player.Write(buf[:n])
		}
	}

	var wg sync.WaitGroup

	scheduleSound := func(key string) {
		// useful to help debug sounds
		// fmt.Printf("Playing sound: %s\n", key)
		wg.Add(1)
		go func() {
			defer wg.Done()
			playSound(key)
		}()
	}

	hook.Register(hook.KeyDown, []string{"A-Z a-z 0-9"}, func(e hook.Event) {

		if keyMap[e.Rawcode] {
			return
		}
		keyMap[e.Rawcode] = true

		if e.Rawcode == ENTER {
			scheduleSound("down_enter")
		} else if e.Rawcode == SPACE {
			scheduleSound("down_space")
		} else {
			scheduleSound(getRandomDownKey())
		}
	})

	hook.Register(hook.KeyUp, []string{"A-Z a-z 0-9"}, func(e hook.Event) {
		if !keyMap[e.Rawcode] {
			return
		}
		keyMap[e.Rawcode] = false

		if e.Rawcode == ENTER {
			scheduleSound("up_enter")
		} else if e.Rawcode == SPACE {
			scheduleSound("up_space")
		} else {
			scheduleSound(getRandomUpKey())
		}
	})

// hook.Register(hook.MouseHold, []string{"mleft"}, func(e hook.Event) {
//     if keyMap[e.Button] {
//         return
//     }
//     keyMap[e.Button] = true
//     scheduleSound("down_mouse")
// })

hook.Register(hook.MouseDown, []string{"mleft"}, func(e hook.Event) {
	if keyMap[e.Button] {
		scheduleSound("up_mouse")
		keyMap[e.Button] = false
		return
	}
	keyMap[e.Button] = true
	scheduleSound("down_mouse")
})

	s := hook.Start()
	<-hook.Process(s)
	wg.Wait()
}
