# Configuration

directory-exporter is configured using a dedicated config file whose file path is passed via the `-config` parameter.

## Example Configuration

```json
{
  "dirs": [
    {
      "frequency": 3600,
      "dir": "/path/to/directory",
      "only_files": true,
      "exclude_files": ["file1.txt", "file2.txt"],
      "include_files": ["file3.txt", "file4.txt"]
    },
    {
      "dir": "/another/directory",
      "only_files": false
    }
  ]
}

```

## Configuration Fields Reference

| Field Name           | Description                                                                                                                                                                    | Data Type        | Optional        |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------|-----------------|
| **Config Struct**    |                                                                                                                                                                                |                  |                 |
| `dirs`               | An array of `DirConfig` objects, each representing a configuration for a specific directory.                                                                                   | Array of objects | No              |
| **DirConfig Struct** |                                                                                                                                                                                |                  |                 |
| `frequency`          | Specifies the frequency in seconds at which the directory should be processed. If omitted, a default value is used.                                                            | Integer          | Yes             |
| `dir`                | Defines the path to the directory that needs to be processed.                                                                                                                  | String           | No              |
| `only_files`         | A boolean flag indicating whether only files within the directory should be processed (if `true`) or if both files and subdirectories should be processed (if `false`).        | Boolean          | No              |
| `exclude_files`      | An array of regular expressions that should be excluded from processing within the directory. Files with matching names will be skipped during processing.                     | Array of strings | Yes             |
| `include_files`      | An array of regular expressions that should be specifically included for processing within the directory. Only files with names matching those in this list will be processed. | Array of strings | Yes             |
