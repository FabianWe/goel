# goel
goel is a reasoner for the Description Logic (DL) EL++ (see [Pushing the EL Envelope Further](http://webont.org/owled/2008dc/papers/owled2008dc_paper_3.pdf`) by Franz Baader, Sebastian Brandt and Carsten Lutz).

More information about the state and usage comming soon, but here are some important installation instructions.

## Installation
The EL++ solver requires that a LP solver is installed. We use [lp_solve](http://lpsolve.sourceforge.net/5.5/) for this, together with the Go wrapper [golp](https://github.com/draffensperger/golp/). However security issues in some newer Go versions broke many of the working cgo projects. At the best of my information right now the last version to really work is go1.9.3. The issue should be fixed in go1.10.1 but that is not released yet. So you need go1.9.3 at the moment.

To install lp_solve and golp correctly first follow the [installation instructions for golp](https://github.com/draffensperger/golp#installation). Note that lp_solve is LGPL licensed and thus not shipped with goel. This means goel uses at run time a copy of the Library already present on the user's computer system.

## Copyright
**The MIT License (MIT)**

Copyright (c) 2016, 2017, 2018 Fabian Wenzelmann

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Third-Party Licenses
Despite from Go the following libraries are used.

 - [golp, MIT licensed](https://github.com/draffensperger/golp#acknowledgements-and-license). See also further details there
 - [lp_solve, LGPL licensed](http://lpsolve.sourceforge.net/5.5/LGPL.htm) and dynamically linked via golp. More details about lp_solve can be found [here](http://lpsolve.sourceforge.net/5.5/).

lp_solve wance to be cited the following way as well:
lpsolve citation data
----------------------
Description     : Open source (Mixed-Integer) Linear Programming system
Language        : Multi-platform, pure ANSI C / POSIX source code, Lex/Yacc based parsing
Official name   : lp_solve (alternatively lpsolve)
Release data    : Version 5.1.0.0 dated 1 May 2004
Co-developers   : Michel Berkelaar, Kjell Eikland, Peter Notebaert
Licence terms   : GNU LGPL (Lesser General Public Licence)
Citation policy : General references as per LGPL
                  Module specific references as specified therein
