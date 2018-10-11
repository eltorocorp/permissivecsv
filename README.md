permissivecsv
=============
PermissiveCSV is a CSV reader that reads non-standard-compliant CSVs. It allows for inconsistencies in the files in exchange for the consumer taking on responsibility for potential mis-reads.

Most CSV readers work from the assumption that the inbound CSV is standards-compliant. As such, typical CSV readers will return errors any time they are unable to parse a record or field due to things like terminator or delimiter inconsistency and field count mismatches.

However, in some use cases, a more permissive reader is desired. PermissiveCSV allows for certain inconsistencies in files by deferring responsibility for data validation to the caller. Instead of trying to enforce standards-compliance, PermissiveCSV instead makes its best judgement about what is happening in a file, and returns the most consistent results possible given those assumptions. Rather than returning errors, PermissiveCSV adjusts its output as best it can for consistency, and returns a Summary of any alterations that were made, after scanning is complete.

Features
========

Sloppy Terminator Support
-------------------------
PermissiveCSV will detect and read CSVs with either unix, DOS, (or a mixture of the two) record terminators.

Inconsistent Field Handling
----------------------------
PermissiveCSV presumes that the number of fields in the first record of the file is correct.
For all subsequent records:
 - If the number of fields is less than expected, blank fields are appended to the result.
 - If the number of fields is more than expected, the the record is truncated.

Header Detection
----------------
PermissiveCSV contains three header detection modes.
1) Assume there is a header.
1) Assume there is no header.
1) Custom detection.

Custom detection is accomplished by supplying a closure function which receives the first two records of the file, and returns true if the comparison between the first two records determines that a header is present.

Partitioning Support
--------------------
PermissiveCSV contains a partition method which takes a desired partition size, and returns a slice of byte offsets which represent the beginning of each partition. Partitioning is guaranteed to work properly even if the file contains a mixture of unix, and DOS terminators.

Errorless Behavior
------------------
PermissiveCSV is errorless. Because it is permissive, it will do everything it can to return data in a consistent format.

In lieu of returning errors, PermissiveCSV has a `Summary()` method, which can be called after a scan is completed. `Summary()` returns an object with statistics about any actions that PermissiveCSV needed to take while reading the file in order to get it into a consistent shape.

For instance, any time a record is appended or truncated as the result of being an unexpected length, the altered record number and operation type (append or truncate) is noted, and reported via the `Summary()` method after the Scan is complete.

