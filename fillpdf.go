package fillpdf

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Form map[string]interface{}

type Options struct {
	Overwrite bool
	Flatten   bool
}

func defaultOptions() Options {
	return Options{
		Overwrite: true,
		Flatten:   true,
	}
}

func Fill(form Form, formPDFFile, destPDFFile string, options ...Options) (err error) {
	opts := defaultOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	formPDFFile, err = filepath.Abs(formPDFFile)
	if err != nil {
		return fmt.Errorf("failed to create the absolute path: %v", err)
	}
	destPDFFile, err = filepath.Abs(destPDFFile)
	if err != nil {
		return fmt.Errorf("failed to create the absolute path: %v", err)
	}

	e, err := exists(formPDFFile)
	if err != nil {
		return fmt.Errorf("failed to check if form PDF file exists: %v", err)
	} else if !e {
		return fmt.Errorf("form PDF file does not exist: '%s'", formPDFFile)
	}

	tmpDir, err := ioutil.TempDir("", "fillpdf-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer func() {
		if errD := os.RemoveAll(tmpDir); errD != nil {
			log.Printf("fillpdf: failed to remove temporary directory '%s': %v", tmpDir, errD)
		}
	}()

	outputFile := filepath.Clean(tmpDir + "/output.pdf")
	fdfFile := filepath.Clean(tmpDir + "/data.fdf")
	if err = createFdfFile(form, fdfFile); err != nil {
		return fmt.Errorf("failed to create fdf form data file: %v", err)
	}

	args := []string{
		"-jar", "mcpdf.jar",
		"fill_form", formPDFFile,
		"data", fdfFile,
		"output", outputFile,
	}

	if opts.Flatten {
		args = append(args, "flatten")
	}

	if err = runCommandInPath(tmpDir, "java", args...); err != nil {
		return fmt.Errorf("mcpdf error: %v", err)
	}

	e, err = exists(destPDFFile)
	if err != nil {
		return fmt.Errorf("failed to check if destination PDF file exists: %v", err)
	} else if e && !opts.Overwrite {
		return fmt.Errorf("destination PDF file already exists: '%s'", destPDFFile)
	} else if e {
		if err = os.Remove(destPDFFile); err != nil {
			return fmt.Errorf("failed to remove destination PDF file: %v", err)
		}
	}

	if err = copyFile(outputFile, destPDFFile); err != nil {
		return fmt.Errorf("failed to copy created output PDF to final destination: %v", err)
	}

	return nil
}

func createFdfFile(form Form, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	fmt.Fprintln(w, fdfHeader)

	var valueStr string
	for key, value := range form {
		valueStr = fmt.Sprintf("%v", value)
		if err != nil {
			return fmt.Errorf("failed to convert string to Latin-1")
		}
		fmt.Fprintf(w, "<< /T (%s) /V (%s)>>\n", key, valueStr)
	}

	fmt.Fprintln(w, fdfFooter)
	return w.Flush()
}

const fdfHeader = `%FDF-1.2
%,,oe"
1 0 obj
<<
/FDF << /Fields [`

const fdfFooter = `]
>>
>>
endobj
trailer
<<
/Root 1 0 R
>>
%%EOF`
