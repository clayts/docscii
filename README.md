docscii
-----
A DocBook to AsciiDoc conversion utility.


Installing
----------
`sudo ./install.sh`

Use
---
`docscii input_dir output_dir`

Or:

`docscii input_file.xml output_dir`

To build books that require publican branding, use instead:

`docscii publican.cfg output_dir`

What it currently does
----------------------
+ Converts even huge documents with complex structures from DocBook
to AsciiDoc basically perfectly.
+ Maintains original file structure
+ Handles basic entities, publican branding, and conditionals
+ Handles customisable "semantic tagging" of various elements
+ Customisable in-line mark up (see `docscii --help` for further information)
