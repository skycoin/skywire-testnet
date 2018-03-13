@echo off

set localCode=localCommit.txt
set remoteHead=remoteHead.txt
set remoteResult=remoteCommit.txt

cd %GOPATH%\src\github.com\skycoin\skywire
call git rev-parse HEAD > %localCode%
set /p local=<%localCode%
del %localCode%
call git rev-parse --abbrev-ref @{u} > %remoteHead%
set /p head=<%remoteHead%
del %remoteHead%
set head=%head:/= %
call git ls-remote %head% > %remoteResult%
set /p remote=<%remoteResult%
del %remoteResult%
set remote=%remote:~0,40%
if %local% neq %remote% (echo "true") else (echo "false")
pause