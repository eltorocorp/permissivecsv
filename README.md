permissivecsv
=============
PermissiveCSV is a CSV reader that reads non-standard-compliant CSVs. It allows for inconsistencies in the files in exchange for the consumer taking on responsibility for potential mis-reads.

Most CSV readers work from the assumption that the inbound CSV is standards-compliant. As such, typical CSV readers will return errors any time they are unable to parse a record or field due to things like terminator or delimiter inconsistency and field count mismatches.

However, in some use cases, a more permissive reader is desired. PermissiveCSV allows for certain inconsistencies in files by deferring responsibility for data validation to the caller. Instead of trying to enforce standards-compliance, PermissiveCSV instead makes its best judgement about what is happening in a file, and returns the most consistent results possible given those assumptions. Rather than returning errors, PermissiveCSV adjusts its output as best it can for consistency, and returns a Summary of any alterations that were made after scanning is complete.

Features
========

Sloppy-Terminator Support
-------------------------
PermissiveCSV will detect and read CSVs with either unix (`\n`), DOS (`\r\n`), inverted DOS (`\n\r`), or carriage return (`\r\`)  record terminators. Furthermore, the terminator is permitted to be inconsistent from record to record.

When scanning a search space for a terminator, PermissiveCSV will select the first non-quoted terminator it encounters using the following order:

1) DOS (`\r\n`)
1) Inverted DOS (`\n\r`)
1) unix (`\n`)
1) Carriage Return (`\r`)

*Terminator evaluation order*

 - PermissiveCSV doesn't make any a priori assumptions about a file-author's intent.
   - Terminators are evaluated solely on the context of the current search space within a file. 
   - To accomplish detection, terminators are evaluated first by length, then by priority within a length. 
   - Terminators can vary in byte-length, and terminators can be composites of eachother (for instance, a DOS terminator is a composite of a unix terminator and a carriage return).
   - The search algorithm gives priority to longer terminators, to ensure that it does not mistakenly select a terminator which is actually a sub-element of a larger composite terminator
     - Example: Selecting `\n` as the terminator, when the terminator was actually `\r\n` 
- Within each terminator length, a priority order is utilized.
  - Example: Between DOS and Inverted DOS, both of which have a length of two, DOS has priority. 
  - Similarly, between unix and Carriage Return, both of which have a length of 1, unix has priority.

*Ignoring Terminators*

 - Terminators that fall anywhere inside a pair of double quotes are ignored. 
 - Outside of double quotes, potential terminators are ignored only if a more likely terminator has been selected for the current record. 
   - Example: If a potential record contains a carriage return and a newline separated by one or more other characters, the newline will be used as the terminator, and the carriage return will be ignored (even though it may not be quoted).

Inconsistent-Record-Length Handling
-----------------------------------
PermissiveCSV presumes that the number of fields in the first record of the file is the intended field count for the entire file.
For all subsequent records:
 - If the number of fields is less than expected, blank fields are appended to the record.
 - If the number of fields is greater than expected, the right-hand side of the record is truncated, such that the number of fields matches the expected field count.

Botched-Quote Handling
----------------------
PermissiveCSV handles two common forms of malformed quotes.
 - Bare quotes: `"grib"flar,foo`
 - Extraneous quotes: `grib,"flar,foo`

Bare and Extraneous quotes are handled similarly. In either
of these conditions, PermissiveCSV will return a record that contains empty
fields. See [Inconsistent Field Handling]() for information about how the
number of fields is deduced.

PermissiveCSV differs from how the standard library `csv.Reader` handles quote
errors. In a `csv.Reader`, if lazy quotes is enabled, `csv.Reader` will push all of the data for the botched record into a single field. By contrast, PermissiveCSV instead returns a set of empty fields. This behavior ensures that the data returned for records with malformed quotes is as consistent as possible across all records who share the same issue. When PermissiveCSV encounters a malformed quote, that encounter, along with the original data, is made immediately available via the Summary method. This reinforces the Summary method as the central source for identifying and acting
upon assumptions that PermissiveCSV has while scanning a file.

Header Detection
----------------
PermissiveCSV contains three header detection modes.
1) Assume there is a header.
1) Assume there is no header.
1) Custom detection.

```
  // Example 1: Setting up a Scanner that assumes there is always a header.
  f, _ := os.Open("somefile.csv")
  s:= permissivecsv.NewScanner(f, permissivecsv.HeaderCheckAssumeHeaderExists)
  s.Scan()
  fmt.Print(s.RecordIsHeader())
  //output: false 
```

```
  // Example 2: Setting up a Scanner that assumes there is no header.
  f, _ := os.Open("somefile.csv")
  s:= permissivecsv.NewScanner(f, permissivecsv.HeaderCheckAssumeNoHeader)
  s.Scan()
  fmt.Print(s.RecordIsHeader()) 
  //output: true
```

```
  // Example 3: Custom detection logic: If first field is "address", it's a header.
  // This is a trivial example. See docs for more information about how the
  // HeaderCheck callback operates.
  headerCheck:= func(firstRecord, secondRecord []string) bool {
    return firstRecord[0] == "address"
  }
  f, _ := os.Open("somefile.csv")
  s:= permissivecsv.NewScanner(f, headerCheck)
  s.Scan()
  fmt.Print(s.RecordIsHeader()) 
  //output: true
```

Partitioning Support
--------------------
PermissiveCSV contains a partition method which takes a desired partition size, and returns a slice of byte offsets which represent the beginning of each partition. Partitioning is guaranteed to work properly even if the file contains a mixture of record terminators.

"Errorless" Behavior
------------------
PermissiveCSV tries hard to avoid returning errors. Because it is permissive, it will do everything it can to return data in a consistent format.

In lieu of returning errors, PermissiveCSV has a `Summary()` method, which can be called after each call to Scan. `Summary()` returns an object with statistics about any actions that PermissiveCSV needed to take while reading the file in order to get it into a consistent shape.

For instance, any time a record is appended or truncated as the result of being an unexpected length, the altered record number and operation type (append or truncate) is noted, and reported via the `Summary()` method after the Scan is complete.

PermissiveCSV has no control over the reader that has been supplied by the caller. If the underlaying reader returns an error, that error will be made available via the `Summary().Err` value. Outside of that, PermissiveCSV will not return any errors so long as the supplied reader continues to supply data.
