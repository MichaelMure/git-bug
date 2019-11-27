## Tracking perf changes

Initial performance:
```
goos: darwin
goarch: amd64
pkg: github.com/gomarkdown/markdown
BenchmarkEscapeHTML-8                           	 2000000	       823 ns/op	       0 B/op	       0 allocs/op
BenchmarkSmartDoubleQuotes-8                    	  300000	      5033 ns/op	    9872 B/op	      56 allocs/op
BenchmarkReferenceAmps-8                        	  100000	     19538 ns/op	   26776 B/op	     150 allocs/op
BenchmarkReferenceAutoLinks-8                   	  100000	     17574 ns/op	   24544 B/op	     132 allocs/op
BenchmarkReferenceBackslashEscapes-8            	   30000	     50977 ns/op	   76752 B/op	     243 allocs/op
BenchmarkReferenceBlockquotesWithCodeBlocks-8   	  200000	      8546 ns/op	   12864 B/op	      65 allocs/op
BenchmarkReferenceCodeBlocks-8                  	  200000	      9000 ns/op	   14912 B/op	      70 allocs/op
BenchmarkReferenceCodeSpans-8                   	  200000	      8856 ns/op	   14992 B/op	      69 allocs/op
BenchmarkReferenceHardWrappedPara-8             	  200000	      6599 ns/op	   11312 B/op	      57 allocs/op
BenchmarkReferenceHorizontalRules-8             	  100000	     15483 ns/op	   23536 B/op	      98 allocs/op
BenchmarkReferenceInlineHTMLAdvances-8          	  200000	      6839 ns/op	   12150 B/op	      62 allocs/op
BenchmarkReferenceInlineHTMLSimple-8            	  100000	     19940 ns/op	   28488 B/op	     117 allocs/op
BenchmarkReferenceInlineHTMLComments-8          	  200000	      7455 ns/op	   13440 B/op	      64 allocs/op
BenchmarkReferenceLinksInline-8                 	  100000	     16425 ns/op	   23664 B/op	     147 allocs/op
BenchmarkReferenceLinksReference-8              	   30000	     54895 ns/op	   66464 B/op	     416 allocs/op
BenchmarkReferenceLinksShortcut-8               	  100000	     17647 ns/op	   23776 B/op	     158 allocs/op
BenchmarkReferenceLiterQuotesInTitles-8         	  200000	      9367 ns/op	   14832 B/op	      95 allocs/op
BenchmarkReferenceMarkdownBasics-8              	   10000	    129772 ns/op	  130848 B/op	     378 allocs/op
BenchmarkReferenceMarkdownSyntax-8              	    3000	    502365 ns/op	  461411 B/op	    1411 allocs/op
BenchmarkReferenceNestedBlockquotes-8           	  200000	      7028 ns/op	   12688 B/op	      64 allocs/op
BenchmarkReferenceOrderedAndUnorderedLists-8    	   20000	     79686 ns/op	  107520 B/op	     374 allocs/op
BenchmarkReferenceStrongAndEm-8                 	  200000	     10020 ns/op	   17792 B/op	      78 allocs/op
BenchmarkReferenceTabs-8                        	  200000	     12025 ns/op	   18224 B/op	      81 allocs/op
BenchmarkReferenceTidyness-8                    	  200000	      8985 ns/op	   14432 B/op	      71 allocs/op
PASS
ok  	github.com/gomarkdown/markdown	45.375s
```

After switching to using interface{} for Node.Data:
```
BenchmarkEscapeHTML-8                           	 2000000	       929 ns/op	       0 B/op	       0 allocs/op
BenchmarkSmartDoubleQuotes-8                    	  300000	      5126 ns/op	    9248 B/op	      56 allocs/op
BenchmarkReferenceAmps-8                        	  100000	     19927 ns/op	   17880 B/op	     154 allocs/op
BenchmarkReferenceAutoLinks-8                   	  100000	     20732 ns/op	   17360 B/op	     141 allocs/op
BenchmarkReferenceBackslashEscapes-8            	   30000	     50267 ns/op	   38128 B/op	     244 allocs/op
BenchmarkReferenceBlockquotesWithCodeBlocks-8   	  200000	      8988 ns/op	   10912 B/op	      67 allocs/op
BenchmarkReferenceCodeBlocks-8                  	  200000	      8611 ns/op	   12256 B/op	      74 allocs/op
BenchmarkReferenceCodeSpans-8                   	  200000	      8256 ns/op	   11248 B/op	      69 allocs/op
BenchmarkReferenceHardWrappedPara-8             	  200000	      6739 ns/op	    9856 B/op	      57 allocs/op
BenchmarkReferenceHorizontalRules-8             	  100000	     15503 ns/op	   15600 B/op	     104 allocs/op
BenchmarkReferenceInlineHTMLAdvances-8          	  200000	      6874 ns/op	   10278 B/op	      62 allocs/op
BenchmarkReferenceInlineHTMLSimple-8            	  100000	     22271 ns/op	   18552 B/op	     121 allocs/op
BenchmarkReferenceInlineHTMLComments-8          	  200000	      8315 ns/op	   10736 B/op	      64 allocs/op
BenchmarkReferenceLinksInline-8                 	  100000	     16155 ns/op	   16912 B/op	     152 allocs/op
BenchmarkReferenceLinksReference-8              	   30000	     52387 ns/op	   38192 B/op	     445 allocs/op
BenchmarkReferenceLinksShortcut-8               	  100000	     17111 ns/op	   16592 B/op	     167 allocs/op
BenchmarkReferenceLiterQuotesInTitles-8         	  200000	      9164 ns/op	   12048 B/op	      97 allocs/op
BenchmarkReferenceMarkdownBasics-8              	   10000	    129262 ns/op	   87264 B/op	     416 allocs/op
BenchmarkReferenceMarkdownSyntax-8              	    3000	    496873 ns/op	  293906 B/op	    1559 allocs/op
BenchmarkReferenceNestedBlockquotes-8           	  200000	      6854 ns/op	   10192 B/op	      64 allocs/op
BenchmarkReferenceOrderedAndUnorderedLists-8    	   20000	     79633 ns/op	   55024 B/op	     447 allocs/op
BenchmarkReferenceStrongAndEm-8                 	  200000	      9637 ns/op	   12176 B/op	      78 allocs/op
BenchmarkReferenceTabs-8                        	  100000	     12164 ns/op	   13776 B/op	      87 allocs/op
BenchmarkReferenceTidyness-8                    	  200000	      8677 ns/op	   11296 B/op	      75 allocs/op
```

Not necessarily faster, but uses less bytes per op (but sometimes more allocs).

After tweaking the API:
```
$ ./s/run-bench.sh

go test -bench=. -test.benchmem
goos: darwin
goarch: amd64
pkg: github.com/gomarkdown/markdown
BenchmarkEscapeHTML-8                           	 2000000	       834 ns/op	       0 B/op	       0 allocs/op
BenchmarkSmartDoubleQuotes-8                    	  300000	      3486 ns/op	    6160 B/op	      27 allocs/op
BenchmarkReferenceAmps-8                        	  100000	     18158 ns/op	   14792 B/op	     125 allocs/op
BenchmarkReferenceAutoLinks-8                   	  100000	     16824 ns/op	   14272 B/op	     112 allocs/op
BenchmarkReferenceBackslashEscapes-8            	   30000	     44066 ns/op	   35040 B/op	     215 allocs/op
BenchmarkReferenceBlockquotesWithCodeBlocks-8   	  200000	      6868 ns/op	    7824 B/op	      38 allocs/op
BenchmarkReferenceCodeBlocks-8                  	  200000	      7157 ns/op	    9168 B/op	      45 allocs/op
BenchmarkReferenceCodeSpans-8                   	  200000	      6663 ns/op	    8160 B/op	      40 allocs/op
BenchmarkReferenceHardWrappedPara-8             	  300000	      4821 ns/op	    6768 B/op	      28 allocs/op
BenchmarkReferenceHorizontalRules-8             	  100000	     13033 ns/op	   12512 B/op	      75 allocs/op
BenchmarkReferenceInlineHTMLAdvances-8          	  300000	      4998 ns/op	    7190 B/op	      33 allocs/op
BenchmarkReferenceInlineHTMLSimple-8            	  100000	     17696 ns/op	   15464 B/op	      92 allocs/op
BenchmarkReferenceInlineHTMLComments-8          	  300000	      5506 ns/op	    7648 B/op	      35 allocs/op
BenchmarkReferenceLinksInline-8                 	  100000	     14450 ns/op	   13824 B/op	     123 allocs/op
BenchmarkReferenceLinksReference-8              	   30000	     52561 ns/op	   35104 B/op	     416 allocs/op
BenchmarkReferenceLinksShortcut-8               	  100000	     15616 ns/op	   13504 B/op	     138 allocs/op
BenchmarkReferenceLiterQuotesInTitles-8         	  200000	      7772 ns/op	    8960 B/op	      68 allocs/op
BenchmarkReferenceMarkdownBasics-8              	   10000	    121436 ns/op	   84176 B/op	     387 allocs/op
BenchmarkReferenceMarkdownSyntax-8              	    3000	    487404 ns/op	  290818 B/op	    1530 allocs/op
BenchmarkReferenceNestedBlockquotes-8           	  300000	      5098 ns/op	    7104 B/op	      35 allocs/op
BenchmarkReferenceOrderedAndUnorderedLists-8    	   20000	     74422 ns/op	   51936 B/op	     418 allocs/op
BenchmarkReferenceStrongAndEm-8                 	  200000	      7888 ns/op	    9088 B/op	      49 allocs/op
BenchmarkReferenceTabs-8                        	  200000	     10061 ns/op	   10688 B/op	      58 allocs/op
BenchmarkReferenceTidyness-8                    	  200000	      7152 ns/op	    8208 B/op	      46 allocs/op
ok  	github.com/gomarkdown/markdown	40.809s
```

After refactoring Renderer:
```
BenchmarkEscapeHTML-8                                    2000000               883 ns/op               0 B/op          0 allocs/op
BenchmarkSmartDoubleQuotes-8                              300000              3717 ns/op            6208 B/op         29 allocs/op
BenchmarkReferenceAmps-8                                  100000             19135 ns/op           14680 B/op        123 allocs/op
BenchmarkReferenceAutoLinks-8                             100000             17142 ns/op           14176 B/op        110 allocs/op
BenchmarkReferenceBackslashEscapes-8                       30000             54616 ns/op           35088 B/op        217 allocs/op
BenchmarkReferenceBlockquotesWithCodeBlocks-8             200000              7993 ns/op            7872 B/op         40 allocs/op
BenchmarkReferenceCodeBlocks-8                            200000              8285 ns/op            9216 B/op         47 allocs/op
BenchmarkReferenceCodeSpans-8                             200000              7684 ns/op            8208 B/op         42 allocs/op
BenchmarkReferenceHardWrappedPara-8                       200000              5595 ns/op            6816 B/op         30 allocs/op
BenchmarkReferenceHorizontalRules-8                       100000             16444 ns/op           12560 B/op         77 allocs/op
BenchmarkReferenceInlineHTMLAdvances-8                    200000              5415 ns/op            7238 B/op         35 allocs/op
BenchmarkReferenceInlineHTMLSimple-8                      100000             19867 ns/op           15512 B/op         94 allocs/op
BenchmarkReferenceInlineHTMLComments-8                    200000              6026 ns/op            7696 B/op         37 allocs/op
BenchmarkReferenceLinksInline-8                           100000             14864 ns/op           13664 B/op        120 allocs/op
BenchmarkReferenceLinksReference-8                         30000             52479 ns/op           34816 B/op        401 allocs/op
BenchmarkReferenceLinksShortcut-8                         100000             15812 ns/op           13472 B/op        135 allocs/op
BenchmarkReferenceLiterQuotesInTitles-8                   200000              7767 ns/op            8880 B/op         68 allocs/op
BenchmarkReferenceMarkdownBasics-8                         10000            131065 ns/op           84048 B/op        386 allocs/op
BenchmarkReferenceMarkdownSyntax-8                          2000            515604 ns/op          289953 B/op       1501 allocs/op
BenchmarkReferenceNestedBlockquotes-8                     200000              5655 ns/op            7152 B/op         37 allocs/op
BenchmarkReferenceOrderedAndUnorderedLists-8               20000             84188 ns/op           51984 B/op        420 allocs/op
BenchmarkReferenceStrongAndEm-8                           200000              8664 ns/op            9136 B/op         51 allocs/op
BenchmarkReferenceTabs-8                                  100000             11110 ns/op           10736 B/op         60 allocs/op
BenchmarkReferenceTidyness-8                              200000              7628 ns/op            8256 B/op         48 allocs/op
ok      github.com/gomarkdown/markdown  40.841s
```

After Node refactor to have Children array:
```
BenchmarkEscapeHTML-8                                    2000000               901 ns/op               0 B/op          0 allocs/op
BenchmarkSmartDoubleQuotes-8                              300000              3905 ns/op            6224 B/op         31 allocs/op
BenchmarkReferenceAmps-8                                  100000             22216 ns/op           15560 B/op        157 allocs/op
BenchmarkReferenceAutoLinks-8                             100000             20335 ns/op           14824 B/op        146 allocs/op
BenchmarkReferenceBackslashEscapes-8                       20000             69174 ns/op           37392 B/op        316 allocs/op
BenchmarkReferenceBlockquotesWithCodeBlocks-8             200000              8443 ns/op            7968 B/op         48 allocs/op
BenchmarkReferenceCodeBlocks-8                            200000              9250 ns/op            9392 B/op         58 allocs/op
BenchmarkReferenceCodeSpans-8                             200000              8515 ns/op            8432 B/op         54 allocs/op
BenchmarkReferenceHardWrappedPara-8                       200000              5738 ns/op            6856 B/op         34 allocs/op
BenchmarkReferenceHorizontalRules-8                       100000             20864 ns/op           13648 B/op         93 allocs/op
BenchmarkReferenceInlineHTMLAdvances-8                    200000              6187 ns/op            7310 B/op         40 allocs/op
BenchmarkReferenceInlineHTMLSimple-8                       50000             23793 ns/op           16128 B/op        114 allocs/op
BenchmarkReferenceInlineHTMLComments-8                    200000              7060 ns/op            7840 B/op         44 allocs/op
BenchmarkReferenceLinksInline-8                           100000             18432 ns/op           14496 B/op        153 allocs/op
BenchmarkReferenceLinksReference-8                         20000             67666 ns/op           37136 B/op        502 allocs/op
BenchmarkReferenceLinksShortcut-8                         100000             19324 ns/op           13984 B/op        162 allocs/op
BenchmarkReferenceLiterQuotesInTitles-8                   200000              8998 ns/op            9320 B/op         83 allocs/op
BenchmarkReferenceMarkdownBasics-8                         10000            160908 ns/op           88152 B/op        518 allocs/op
BenchmarkReferenceMarkdownSyntax-8                          2000            707160 ns/op          303801 B/op       2044 allocs/op
BenchmarkReferenceNestedBlockquotes-8                     200000              6740 ns/op            7248 B/op         45 allocs/op
BenchmarkReferenceOrderedAndUnorderedLists-8               10000            115808 ns/op           55052 B/op        626 allocs/op
BenchmarkReferenceStrongAndEm-8                           100000             10540 ns/op            9416 B/op         72 allocs/op
BenchmarkReferenceTabs-8                                  100000             13171 ns/op           10968 B/op         77 allocs/op
BenchmarkReferenceTidyness-8                              200000              8903 ns/op            8404 B/op         62 allocs/op
PASS
ok      github.com/gomarkdown/markdown  43.477s
```
It's slower (but opens up possibilities for further improvements).

After refactoring to make ast.Node a top-level thing.
```
BenchmarkEscapeHTML-8                                    2000000               829 ns/op               0 B/op          0 allocs/op
BenchmarkSmartDoubleQuotes-8                              300000              3998 ns/op            6192 B/op         31 allocs/op
BenchmarkReferenceAmps-8                                   50000             27389 ns/op           15480 B/op        153 allocs/op
BenchmarkReferenceAutoLinks-8                              50000             23106 ns/op           14656 B/op        137 allocs/op
BenchmarkReferenceBackslashEscapes-8                       10000            112435 ns/op           36696 B/op        315 allocs/op
BenchmarkReferenceBlockquotesWithCodeBlocks-8             200000              9227 ns/op            7856 B/op         46 allocs/op
BenchmarkReferenceCodeBlocks-8                            200000             10469 ns/op            9248 B/op         54 allocs/op
BenchmarkReferenceCodeSpans-8                             200000             10522 ns/op            8368 B/op         54 allocs/op
BenchmarkReferenceHardWrappedPara-8                       200000              6354 ns/op            6784 B/op         34 allocs/op
BenchmarkReferenceHorizontalRules-8                        50000             32393 ns/op           13952 B/op         87 allocs/op
BenchmarkReferenceInlineHTMLAdvances-8                    200000              6894 ns/op            7238 B/op         40 allocs/op
BenchmarkReferenceInlineHTMLSimple-8                       50000             32942 ns/op           15864 B/op        110 allocs/op
BenchmarkReferenceInlineHTMLComments-8                    200000              8181 ns/op            7776 B/op         44 allocs/op
BenchmarkReferenceLinksInline-8                           100000             21679 ns/op           14400 B/op        148 allocs/op
BenchmarkReferenceLinksReference-8                         20000             83928 ns/op           36688 B/op        473 allocs/op
BenchmarkReferenceLinksShortcut-8                         100000             22053 ns/op           13872 B/op        153 allocs/op
BenchmarkReferenceLiterQuotesInTitles-8                   100000             10784 ns/op            9296 B/op         81 allocs/op
BenchmarkReferenceMarkdownBasics-8                          5000            237097 ns/op           87760 B/op        480 allocs/op
BenchmarkReferenceMarkdownSyntax-8                          1000           1465402 ns/op          300769 B/op       1896 allocs/op
BenchmarkReferenceNestedBlockquotes-8                     200000              7461 ns/op            7152 B/op         45 allocs/op
BenchmarkReferenceOrderedAndUnorderedLists-8                5000            212256 ns/op           53724 B/op        553 allocs/op
BenchmarkReferenceStrongAndEm-8                           100000             13018 ns/op            9264 B/op         72 allocs/op
BenchmarkReferenceTabs-8                                  100000             15005 ns/op           10752 B/op         71 allocs/op
BenchmarkReferenceTidyness-8                              200000             10308 ns/op            8292 B/op         58 allocs/op
PASS
ok      github.com/gomarkdown/markdown  42.176s
```
