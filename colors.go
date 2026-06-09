package main

// colors holds the 256-color palette indices used for cycling the rainbow
// foreground in non-mono mode. The numeric values map to the standard
// xterm 256-color cube and are emitted via the SGR escape `\x1b[38;5;<n>m`.
var colors = []int{
	210, // peach
	222, // orange
	120, // green
	123, // cyan
	111, // blue
	134, // purple
	177, // pink
	207, // fuschia
	206, // magenta
	204, // red
}
