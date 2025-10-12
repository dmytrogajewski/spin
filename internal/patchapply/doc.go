// Package patchapply provides file modification via structured patches.
//
// The package implements Spin's custom patch format designed for AI models
// to generate correct, safe file modifications. Unlike standard diff formats,
// Spin's format is simple, unambiguous, and resistant to generation errors.
//
// # Patch Format
//
// A patch consists of operations enclosed in Begin/End markers:
//
//	*** Begin Patch
//	*** Add File: path/to/file.txt
//	+line 1
//	+line 2
//	*** End Patch
//
// Supported operations:
//   - Add File: Create new file with content
//   - Delete File: Remove existing file
//   - Update File: Modify file with hunks
//
// # Security
//
// All file paths are validated using pkg/pathutil to prevent:
//   - Path traversal attacks (../../etc/passwd)
//   - Absolute path injection (/etc/passwd)
//   - Symlink escapes (link_to_etc/passwd)
//
// # Usage
//
//	parser := patchapply.NewParser(patchText)
//	patch, err := parser.Parse()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, op := range patch.Operations {
//	    switch op := op.(type) {
//	    case *patchapply.AddFile:
//	        fmt.Printf("Add: %s\n", op.FilePath)
//	    case *patchapply.DeleteFile:
//	        fmt.Printf("Delete: %s\n", op.FilePath)
//	    case *patchapply.UpdateFile:
//	        fmt.Printf("Update: %s\n", op.FilePath)
//	    }
//	}
//
// # Error Handling
//
// Parse errors include line numbers for debugging:
//
//	line 5: invalid path "/etc/passwd": absolute paths not allowed
//
// This helps AI models understand and correct their patch generation.
package patchapply
