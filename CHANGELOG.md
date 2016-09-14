# Update

## [1.0.9](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.9)
버그 수정

## [1.0.8](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.8)
goinside 패키지 업데이트를 적용한 바이너리입니다.

## [1.0.7](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.7)
좀 더 최적화 하였습니다.

## [1.0.6](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.6)
일부 게시물에서 이미지 파싱이 제대로 이루어지지 않는 문제를 goinside에서 수정하였습니다. 

프로그램의 사용 방법을 변경하였습니다. 기존에 -gall 인자로 URL을 전달했던 것을, -gall 인자 혹은 -url 인자를 사용하는 것으로 변경하였습니다. 이제 -gall 인자는 갤러리의 ID를 받으며, -url 인자는 기존처럼 URL을 받습니다.

## [1.0.5](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.5)
goinside API 변경에 따른 코드 변경 및 일부 로그 수정.

## [1.0.4](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.4)
img가 아닌 다른 html element까지 image로 파싱하는 문제를 goinside에서 수정하였습니다.

## [1.0.3](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.3)
이미지가 저장되는 디렉토리 구조를 변경하였습니다. 기존에 `image` 디렉토리에 모두 저장되던 방식 대신, `images` 디렉토리에 해당 갤러리의 id를 이름으로 가지는 하위 디렉토리가 생성되고, 이곳에 해당 갤러리의 이미지를 저장하게 됩니다. 예를 들어 프로그래밍 갤러리의 경우, `images/programming` 디렉토리에 이미지가 저장됩니다.

## [1.0.2](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.2)
일부 게시물에서 이미지가 처음 하나만 파싱되는 오류를 goinside 패키지에서 수정하였습니다. 

## [1.0.1](https://github.com/geeksbaek/goinside-image-crawler/releases/tag/1.0.1)
프로그램이 다시 시작할 때마다 `image` 디렉토리의 이미지들을 다시 hashing하는 일을 막기 위해 hash strirng을 이미지 파일 이름으로 사용하도록 변경하였습니다. 기존에 `image` 디렉토리에 존재하는 파일 중, 확장자를 제외한 파일 이름의 길이가 40(sha-1 hash string length)이 아닌 파일은 hash string이 적용되지 않은 파일로 간주하고 hashing 후 해당 파일 이름을 hash string으로 변경하는 작업을 거치게 됩니다.