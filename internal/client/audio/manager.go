// internal/client/audio/manager.go
package audio

import (
	"fmt"
	"log"
	"sync"
)

// Manager gère tous les sons du jeu
type Manager struct {
	sounds      map[string]*Sound
	musicVolume float64
	sfxVolume   float64
	enabled     bool
	mu          sync.RWMutex
}

// Sound représente un fichier audio
type Sound struct {
	Name     string
	FilePath string
	IsLoaded bool
}

// NewManager crée un nouveau gestionnaire audio
func NewManager() *Manager {
	return &Manager{
		sounds:      make(map[string]*Sound),
		musicVolume: 0.7,
		sfxVolume:   0.8,
		enabled:     true,
	}
}

// LoadSound charge un son en mémoire
func (m *Manager) LoadSound(name, filepath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Créer le son
	sound := &Sound{
		Name:     name,
		FilePath: filepath,
		IsLoaded: true,
	}

	m.sounds[name] = sound
	log.Printf("🔊 Loaded sound: %s", name)
	return nil
}

// PlaySound joue un son
func (m *Manager) PlaySound(name string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return nil
	}

	snd, exists := m.sounds[name]
	if !exists {
		return fmt.Errorf("sound not found: %s", name)
	}

	if !snd.IsLoaded {
		return fmt.Errorf("sound not loaded: %s", name)
	}

	// Jouer le son avec le volume SFX
	log.Printf("🔊 Playing sound: %s (volume: %.0f%%)", name, m.sfxVolume*100)
	// TODO: Implémenter la lecture audio réelle avec beep ou portaudio

	return nil
}

// PlayMusic joue de la musique de fond
func (m *Manager) PlayMusic(name string, loop bool) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return nil
	}

	_, exists := m.sounds[name]
	if !exists {
		return fmt.Errorf("music not found: %s", name)
	}

	log.Printf("🎵 Playing music: %s (loop: %v, volume: %.0f%%)", name, loop, m.musicVolume*100)
	// TODO: Implémenter la lecture de musique en boucle

	return nil
}

// StopMusic arrête la musique
func (m *Manager) StopMusic() {
	log.Println("⏹️ Music stopped")
	// TODO: Implémenter l'arrêt de la musique
}

// SetMusicVolume définit le volume de la musique (0.0 - 1.0)
func (m *Manager) SetMusicVolume(volume float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}

	m.musicVolume = volume
	log.Printf("🎵 Music volume: %.0f%%", volume*100)
}

// SetSFXVolume définit le volume des effets sonores (0.0 - 1.0)
func (m *Manager) SetSFXVolume(volume float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}

	m.sfxVolume = volume
	log.Printf("🔊 SFX volume: %.0f%%", volume*100)
}

// Enable active le son
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
	log.Println("🔊 Audio enabled")
}

// Disable désactive le son
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	m.StopMusic()
	log.Println("🔇 Audio disabled")
}

// IsEnabled retourne l'état du son
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// LoadAllSounds charge tous les sons du jeu
func (m *Manager) LoadAllSounds() error {
	sounds := map[string]string{
		"dice_roll":        "assets/sounds/dice_roll.mp3",
		"token_move":       "assets/sounds/token_move.mp3",
		"token_capture":    "assets/sounds/token_capture.mp3",
		"your_turn":        "assets/sounds/your_turn.mp3",
		"victory":          "assets/sounds/victory.mp3",
		"defeat":           "assets/sounds/defeat.mp3",
		"button_click":     "assets/sounds/button_click.mp3",
		"background_music": "assets/sounds/background_music.mp3",
	}

	for name, path := range sounds {
		if err := m.LoadSound(name, path); err != nil {
			log.Printf("⚠️ Failed to load sound %s: %v", name, err)
			// Continue même si un son ne charge pas
		}
	}

	return nil
}

// Cleanup libère les ressources audio
func (m *Manager) Cleanup() {
	m.StopMusic()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sounds = make(map[string]*Sound)
	log.Println("🧹 Audio cleanup completed")
}
