//go:build darwin

package pdf

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
)

const osascriptPath = "/usr/bin/osascript"

// macOS Vision フレームワークを使った OCR 用 AppleScript コード
// use scripting additions + use framework で ObjC ブリッジを利用する（-l AppleScriptObjC は不要）
const visionOCRScript = `
use scripting additions
use framework "Foundation"
use framework "Quartz"
use framework "Vision"
use framework "AppKit"

on ocrPDF(filePath, dpi)
	set CA to current application
	set request to CA's VNRecognizeTextRequest's alloc's init
	request's setRecognitionLevel:(CA's VNRequestTextRecognitionLevelAccurate)
	request's setUsesLanguageCorrection:true
	request's setRecognitionLanguages:{"ja", "en"}

	set doc to CA's PDFDocument's alloc's initWithURL:(CA's NSURL's fileURLWithPath:filePath)
	if doc is missing value then
		error "Failed to open PDF: " & filePath
	end if
	set pageCount to doc's pageCount as integer
	set resultTexts to CA's NSMutableArray's new()
	repeat with i from 1 to pageCount
		set scaleFactor to (dpi / (72.0 * (CA's NSScreen's mainScreen's backingScaleFactor)))
		set pdfImageRep to (CA's NSPDFImageRep's imageRepWithData:((doc's pageAtIndex:(i - 1))'s dataRepresentation))
		set originalSize to pdfImageRep's |bounds|
		set originalWidth to CA's NSWidth(originalSize)
		set originalHeight to CA's NSHeight(originalSize)
		set scaledSize to CA's NSMakeSize(originalWidth * scaleFactor, originalHeight * scaleFactor)
		set targetRect to CA's NSMakeRect(0, 0, scaledSize's width, scaledSize's height)
		set image to (CA's NSImage's alloc's initWithSize:(targetRect's item 2))
		image's lockFocus()
		CA's NSColor's whiteColor's |set|()
		(CA's NSBezierPath's fillRect:targetRect)
		(pdfImageRep's drawInRect:targetRect)
		image's unlockFocus()
		set tiff to image's TIFFRepresentation
		set ocrText to my ocrTIFF(tiff, request)
		(resultTexts's addObject:ocrText)
	end repeat
	return (resultTexts's componentsJoinedByString:linefeed) as text
end ocrPDF

on ocrTIFF(tiff, request)
	set CA to current application
	set resultTexts to CA's NSMutableArray's new()
	set requestHandler to (CA's VNImageRequestHandler's alloc's initWithData:tiff options:(missing value))
	(requestHandler's performRequests:{request} |error|:(missing value))
	set results to request's results()
	repeat with aResult in results
		(resultTexts's addObject:(((aResult's topCandidates:1)'s objectAtIndex:0)'s |string|()))
	end repeat
	return (resultTexts's componentsJoinedByString:linefeed) as text
end ocrTIFF

on run argv
	set filePath to item 1 of argv
	set dpi to 300
	return my ocrPDF(filePath, dpi)
end run
`

// Extractor は macOS の Vision フレームワーク (osascript 経由) を使ってPDFからテキストを抽出する
type Extractor struct {
	path string // 互換性のため保持（macOS では未使用）
}

// NewExtractor は Extractor を生成する
// macOS では osascript + Vision フレームワークを使うため、pdftotext のパスは不要
func NewExtractor(path string) *Extractor {
	return &Extractor{path: path}
}

// IsAvailable は macOS では常に true を返す（osascript は標準搭載）
func (e *Extractor) IsAvailable() bool {
	return true
}

// Path は互換性のために設定されているパスを返す
func (e *Extractor) Path() string {
	return e.path
}

// ExtractText は osascript + Vision フレームワークでPDFからテキストを抽出する
func (e *Extractor) ExtractText(pdfPath string) (string, error) {
	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	cmd := exec.Command(osascriptPath, "-e", visionOCRScript, absPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("vision OCR failed: %w (stderr: %s)", err, stderr.String())
	}

	text := stdout.String()
	if text == "" {
		return "", fmt.Errorf("vision OCR returned empty text for %s", pdfPath)
	}

	return text, nil
}
