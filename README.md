# goinside-image-crawler 
[![Build Status](https://travis-ci.org/geeksbaek/goinside-image-crawler.svg?branch=master)](https://travis-ci.org/geeksbaek/goinside-image-crawler)

goinside-image-crawler는 [goinside](https://github.com/geeksbaek/goinside) 기반으로 만든, 특정 갤러리의 첫 페이지에 있는 이미지들을 실시간으로 수집하는 프로그램입니다.

프로그램을 실행한 경로에 `images`라는 디렉토리가 생성되며, 입력된 갤러리의 id 이름으로 하위 디렉토리가 이곳에 생성됩니다. 해당 갤러리에서 수집된 이미지는 이 하위 디렉토리로에 저장됩니다. ex) `images/programming`

디시인사이드에 정상적인 방법으로 업로드 된 이미지만 수집됩니다. 해쉬로 이미지 중복을 검사하며, 해당 디렉토리 내에 있는 이미지와 중복되는 이미지는 저장하지 않습니다. 

파일 이름은 해당 이미지의 해쉬 값이므로 정상적인 중복 검사를 위해 변경하지 않는 것을 추천합니다. 만약 수정을 해야 한다면 다른 경로로 파일을 복사한 뒤 수정하여야 합니다.

## Install
```
$ go get -u github.com/geeksbaek/goinside-image-crawler
```
go get 명령어로 직접 패키지를 인스톨해서 빌드하는 대신, [여기](https://github.com/geeksbaek/goinside-image-crawler/releases/latest)에서 실행 파일을 직접 다운로드 할 수 있습니다.

## Usage
```
// using url
$ goinside-image-crawler.exe -url http://gall.dcinside.com/board/lists/?id=programming

// or, using gall id
$ goinside-image-crawler.exe -gall programming
```

Jongyeol Baek <geeksbaek@gmail.com>
