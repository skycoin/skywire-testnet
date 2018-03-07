@echo off

echo "update skywire"
cd %GOPATH%\src\github.com\skycoin\skywire
call git reset --hard
call git clean -f -d
call git pull origin master

call %GOPATH%\src\github.com\skycoin\skywire\static/script/win/start.bat