# goinside-image-crawler

goinside-image-crawler는 디시인사이드 특정 갤러리의 글에 첨부된 이미지를 실시간으로 수집하는 프로그램입니다. 
프로그램을 실행한 경로에 `image`라는 디렉토리가 생성되며, 수집된 이미지는 이 경로에 저장됩니다.

해쉬로 이미지 중복을 검사하며, 해당 디렉토리 내에 있는 이미지와 중복되는 이미지는 저장하지 않습니다.

```
// install
go get github.com/geeksbaek/goinside-image-crawler

// usage
goinside-image-crawler.exe -gall http://gall.dcinside.com/board/lists/?id=programming
```

### update 2016-08-05
프로그램이 다시 시작할 때마다 `image` 디렉토리의 이미지들을 다시 hashing하는 일을 막기 위해 hash strirng을 이미지 파일 이름으로 사용하도록 변경하였습니다. 기존에 `image` 디렉토리에 존재하는 파일 중, 확장자를 제외한 파일 이름의 길이가 40(sha-1 hash string length)이 아닌 파일은 hash string이 적용되지 않은 파일로 간주하고 hashing 후 해당 파일 이름을 hash string으로 변경하는 작업을 거치게 됩니다.

Jongyeol Baek <geeksbaek@gmail.com>
