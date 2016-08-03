# goinside-image-crawler

goinside-image-crawler는 디시인사이드 특정 갤러리의 글에 첨부된 이미지를 실시간으로 수집하는 프로그램입니다. 
프로그램을 실행한 경로에 `image`라는 디렉토리가 생성되며, 수집된 이미지는 이 경로에 저장됩니다.

md5 체크섬으로 이미지 중복을 검사하며, 해당 디렉토리 내에 있는 이미지와 중복되는 이미지는 저장하지 않습니다.

```
// install
go get github.com/geeksbaek/goinside-image-crawler

// usage
goinside-image-crawler.exe -gall http://gall.dcinside.com/board/lists/?id=programming
```

Jongyeol Baek <geeksbaek@gmail.com>
