
# Description
This file lists sample EXT FS images with various corruption types, taken from e2fsprogs test suite (names preserved). The table shows the exit codes for different `e2fsck` options executed consecutively on the same image.

# Highlights
- code 0,1,2 from the repair in preen mode (column 3) is not indicative of success, we cannot tell if any or all errors were fixed, see `f_jnl_etb_alloc_fail`.
- code 4 from the the repair in preen mode (column 3) is not indicative of any errors fixed or even found, see `f_first_meta_bg_too_big`.

# Usage
A subset of these images is used in `TestEXTChecker_WithRealImages` to validate that the checker does not everreact to the non-zero exit codes from `e2fsck -p` and also does not treat zero exit codes as success.

```
-n -nf -p  -nf
---------------------------------------------------------------
12	12	0	0	d_corrupt_journal_nr_users
0	0	0	0	d_inline_dump
4	4	1	0	f_bad_bbitmap
0	12	0	12	f_badbblocks
0	4	0	4	f_badcluster
12	12	4	12	f_baddir
4	4	4	4	f_bad_disconnected_inode
12	12	1	0	f_badjourblks
12	12	8	12	f_clear_orphan_file
8	8	8	8	f_crashdisk
0	8	0	8	f_extent_too_deep
12	12	12	12	f_ext_journal
0	0	1	0	f_extra_journal
0	0	4	0	f_first_meta_bg_too_big
8	8	4	4	f_h_badroot
8	8	4	8	f_illitable
32	12	4	12	f_illitable_flexbg
0	4	0	0	f_jnl_64bit
12	12	1	4	f_jnl_etb_alloc_fail
32	32	4	8	f_misstable
4	4	0	0	f_orphan
0	0	8	0	f_zero_inode_size
4	4	8	4	f_zero_super
```
