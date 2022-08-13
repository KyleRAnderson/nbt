package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

func createHelperFile(dir string, constants []string) error {
	const helperFileBasename = "helper." + generatedFileExt + ".go"
	file, err := os.Create(filepath.Join(dir, helperFileBasename))
	if err != nil {
		return fmt.Errorf(`generator.createHelperFile: failed to open file: %w`, err)
	}
	defer file.Close()
	buf := bufio.NewWriter(file)
	if err := generateHelper(buf, constants); err != nil {
		return err
	}
	buf.Flush()
	if err := reformatFile(file); err != nil {
		return fmt.Errorf(`generator.createHelperFile: failed to autoformat result: %w`, err)
	}
	return nil
}

/*
Generates the helper file given the names of the necessary constants.
The names should be in a deterministic order such that repetitive calls to the
generator always yield the same output.
*/
func generateHelper(writer *bufio.Writer, constants []string) error {
	if err := writeGeneratedHeader(writer); err != nil {
		return fmt.Errorf(`generator.generateHelper: failed to add generated message header: %w`, err)
	}
	if _, err := writer.WriteString("package main\n"); err != nil {
		return fmt.Errorf(`generator.generateHelper: failed to write top line: %w`, err)
	}
	if len(constants) > 0 {
		writer.WriteString("const (\n")
		for _, constName := range constants {
			if _, err := writer.WriteString("\t" + constName + " uint = iota\n"); err != nil {
				return fmt.Errorf(`generator.generateHelper: failed to write:  %w`, err)
			}
		}
		if _, err := writer.WriteString(")"); err != nil {
			return fmt.Errorf(`generator.generateHelper: failed to write: %w`, err)
		}
	}
	return nil
}