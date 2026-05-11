// Package cmd implements the sampleslice CLI using cobra and viper.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sampleslice/config"
	"sampleslice/internal/app"
)

var (
	cfgFile string
	v       = viper.New()
)

var rootCmd = &cobra.Command{
	Use:   "sampleslice",
	Short: "SampleSlice slices audio files into MPC programs",
	Long: `SampleSlice is a command-line tool that detects transients in audio files
and automatically creates MPC (Music Production Center) programs with individual
slice samples.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	voiceMode := config.VoiceModeDrum
	if v.GetString("voice-mode") == "poly" {
		voiceMode = config.VoiceModePoly
	}

	priority := uint8(7) // drum default
	if p := v.GetUint("priority"); p > 0 {
		priority = uint8(p)
	} else if voiceMode == config.VoiceModePoly {
		priority = 3
	}

	envelope := config.Envelope{}
	if raw := v.GetString("envelope"); raw != "" {
		parsed, err := config.ParseEnvelope(raw)
		if err != nil {
			return fmt.Errorf("invalid --envelope value: %w", err)
		}
		envelope = parsed
	} else if preset := v.GetString("envelope-preset"); preset != "" {
		if e, ok := config.GetEnvelopePreset(preset); ok {
			envelope = e
		} else {
			fmt.Fprintf(os.Stderr, "Warning: unknown envelope preset %q, using defaults\n", preset)
		}
	}

	var noteMap []string
	if nm := v.GetString("note-map"); nm != "" {
		noteMap = strings.Split(nm, ",")
	}

	cfg := &config.Config{
		InputPath:           v.GetString("input"),
		OutputDir:           v.GetString("output"),
		Format:              config.OutputFormat(v.GetString("format")),
		ProgramName:         v.GetString("name"),
		RootNote:            v.GetInt("root"),
		PrePadding:          v.GetInt("pre"),
		PostPadding:         v.GetInt("post"),
		GMMap:               v.GetBool("gm-map"),
		NoteMap:             noteMap,
		SimilarityThreshold: v.GetFloat64("dedupe"),
		Normalize:           v.GetBool("normalize"),
		AutoTrim:            v.GetBool("auto-trim"),
		TrimNoiseFloor:      v.GetFloat64("trim-floor"),
		BPM:                 v.GetFloat64("bpm"),
		LoopBars:            v.GetInt("loop-bars"),
		BeatsPerBar:         v.GetInt("beats-per-bar"),
		ReportFormat:        v.GetString("report"),
		Detection: config.DetectionConfig{
			Sensitivity:  v.GetFloat64("sensitivity"),
			MinInterval:  v.GetInt("min-interval"),
			WindowSizeMs: v.GetInt("window-size"),
		},
		KRZ: config.KRZConfig{
			Version:        uint16(v.GetUint32("krz-version")),
			Compress:       v.GetBool("krz-compress"),
			VoiceMode:      voiceMode,
			Priority:       priority,
			Stereo:         v.GetBool("stereo"),
			Envelope:       envelope,
			EnvelopePreset: v.GetString("envelope-preset"),
		},
	}

	return app.Run(cfg)
}

// mustBindFlag binds a viper key to a cobra flag. Panics on programmer error
// (wrong flag name), which surfaces immediately at startup.
func mustBindFlag(key, flagName string) {
	if err := v.BindPFlag(key, rootCmd.Flags().Lookup(flagName)); err != nil {
		panic(fmt.Sprintf("binding viper key %q to flag %q: %v", key, flagName, err))
	}
}

// Execute runs the root cobra command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sampleslice.yaml)")

	rootCmd.Flags().StringP("input", "i", "", "Path to input WAV file (required)")
	rootCmd.Flags().StringP("output", "o", "", "Path to output directory (default: same dir as input)")
	rootCmd.Flags().StringP("name", "n", "SampleSlice", "Name of the MPC program")
	rootCmd.Flags().IntP("root", "r", 48, "Root MIDI note number (default: 48 = C3)")
	rootCmd.Flags().Float64P("sensitivity", "s", 0.5, "Transient detection sensitivity 0.0-1.0 (default: 0.5)")
	rootCmd.Flags().IntP("min-interval", "m", 50, "Minimum interval between transients in ms (default: 50)")
	rootCmd.Flags().Int("window-size", 0, "Energy window size in ms for transient detection; 0 = default (10ms)")
	rootCmd.Flags().IntP("pre", "p", 20, "Pre-padding in ms before each transient (default: 20)")
	rootCmd.Flags().IntP("post", "P", 200, "Post-padding in ms after each transient (default: 200)")
	rootCmd.Flags().StringP("format", "f", "mpc", "Output format: mpc, krz, both, or xpm (default: mpc)")
	rootCmd.Flags().Uint("krz-version", 2000, "KRZ file version number (default: 2000)")
	rootCmd.Flags().Bool("krz-compress", false, "Use ADPCM compression for KRZ samples (default: false)")
	rootCmd.Flags().String("voice-mode", "drum", "KRZ voice mode: drum or poly (default: drum)")
	rootCmd.Flags().Uint("priority", 0, "KRZ voice priority 1-8 (default: 7 for drum, 3 for poly)")
	rootCmd.Flags().Bool("stereo", false, "Enable stereo voice mode for poly patches (default: false)")
	rootCmd.Flags().String("envelope", "", "Raw envelope values: attack,decay1,level1,decay2,level2,decay3,level3,sustain,release (0-255 each)")
	rootCmd.Flags().String("envelope-preset", "", "Envelope preset: drum, perc, pad, key (overridden by --envelope if both set)")
	rootCmd.Flags().Bool("gm-map", false, "Use General MIDI drum mapping (channel 10, standard GM drum notes)")
	rootCmd.Flags().String("note-map", "", "Custom note assignments as comma-separated 'index=note' pairs, e.g. '0=36,1=42,2=46'")
	rootCmd.Flags().Float64("dedupe", 0, "Similarity threshold for deduplication 0.0-1.0 (0=off, 0.95=remove near-duplicates)")
	rootCmd.Flags().Bool("normalize", false, "Peak-normalize each slice to 0 dBFS before export (default: false)")
	rootCmd.Flags().Bool("auto-trim", false, "Strip inaudible leading/trailing silence from each slice (default: false)")
	rootCmd.Flags().Float64("trim-floor", 0.001, "Noise floor for auto-trim in linear amplitude (default: 0.001 ≈ -60 dBFS)")
	rootCmd.Flags().String("report", "", "Write a transient report: json or csv (default: disabled)")
	rootCmd.Flags().Float64("bpm", 0, "Tempo in BPM for beat-grid slicing; 0 = use transient detection (default: 0)")
	rootCmd.Flags().Int("loop-bars", 1, "Number of bars per slice in tempo-grid mode (default: 1)")
	rootCmd.Flags().Int("beats-per-bar", 4, "Time signature numerator for tempo-grid mode (default: 4)")

	mustBindFlag("input", "input")
	mustBindFlag("output", "output")
	mustBindFlag("format", "format")
	mustBindFlag("name", "name")
	mustBindFlag("root", "root")
	mustBindFlag("sensitivity", "sensitivity")
	mustBindFlag("min-interval", "min-interval")
	mustBindFlag("window-size", "window-size")
	mustBindFlag("pre", "pre")
	mustBindFlag("post", "post")
	mustBindFlag("krz-version", "krz-version")
	mustBindFlag("krz-compress", "krz-compress")
	mustBindFlag("voice-mode", "voice-mode")
	mustBindFlag("priority", "priority")
	mustBindFlag("stereo", "stereo")
	mustBindFlag("envelope", "envelope")
	mustBindFlag("envelope-preset", "envelope-preset")
	mustBindFlag("gm-map", "gm-map")
	mustBindFlag("note-map", "note-map")
	mustBindFlag("dedupe", "dedupe")
	mustBindFlag("normalize", "normalize")
	mustBindFlag("auto-trim", "auto-trim")
	mustBindFlag("trim-floor", "trim-floor")
	mustBindFlag("report", "report")
	mustBindFlag("bpm", "bpm")
	mustBindFlag("loop-bars", "loop-bars")
	mustBindFlag("beats-per-bar", "beats-per-bar")

	// Static completions for enumerated flags.
	_ = rootCmd.RegisterFlagCompletionFunc("format",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"mpc", "krz", "both", "xpm"}, cobra.ShellCompDirectiveNoFileComp
		})
	_ = rootCmd.RegisterFlagCompletionFunc("voice-mode",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"drum", "poly"}, cobra.ShellCompDirectiveNoFileComp
		})
	_ = rootCmd.RegisterFlagCompletionFunc("envelope-preset",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"drum", "perc", "pad", "key"}, cobra.ShellCompDirectiveNoFileComp
		})
	_ = rootCmd.RegisterFlagCompletionFunc("report",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"json", "csv"}, cobra.ShellCompDirectiveNoFileComp
		})
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolving home directory: %w", err)
		}
		v.AddConfigPath(home)
		v.AddConfigPath(".")
		v.SetConfigName(".sampleslice")
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("SAMPLESLICE")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", v.ConfigFileUsed())
	}
	return nil
}
