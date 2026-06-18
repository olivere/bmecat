/*
Package bmecat2005 supports reading and writing BMEcat files
according to the 2005 version of the specification (also known as
BMEcat 2.0).

BMEcat is a standard for electronic product catalogs.
See http://www.bmecat.org/ for details about the format.
The specifications are available at
http://www.bme.de/initiativen/bmecat/download/.

The API mirrors the bmecat12 package: reading is streaming and
handler-based via Reader.Do, and writing is driven by a CatalogWriter
passed to Writer.Do. The structural differences from version 1.2 are
mostly element renames (ARTICLE becomes PRODUCT, EAN becomes
INTERNATIONAL_PID, MANUFACTURER_AID becomes MANUFACTURER_PID, and so
on), so the two packages are intentionally learnable as one.
*/
package bmecat2005
