# archiver
Simple program to generate archives from a src folder and save it in a target folder. Additionally it has an option to delete old archives with the same content. It always keeps the most current (as determined by modification time) archive of those with the same content.

```
Usage of ./archiver:
  -delete-old
    	Delete older archives with same content
  -format string
    	Archive name format (default "workspace-2006-01-02T15:04:05Z07:00.tar")
  -src string
    	Path to directory to archive
  -target string
    	Path to storage directory
```
