// Package buffer implements the pure, grapheme-accurate document model for Flourish.
//
// Coordinates are 0-based (Row, GraphemeCol) in grapheme clusters.
// Ranges are half-open selections in document coordinates: [Start, End).
package buffer
