# bumblebee - 이미지 처리 서버

`bumblebee`는 쿠뮤의 이미지 처리 작업을 담당하는 마이크로서비스이다. Go의 Worker pool pattern이나 pipeline pattern, Fan-in Fan-out pattern과 같은 여러 Concurrency pattern(동시성 패턴)들을 적용시켜보고자했으며 기존의 단순한 함수 호출 형태와는 다른 구조를 갖고있다.

## Concurrency pattern in Go

Go 언어의 큰 특징 중 하나는 다양한 Concurrency pattern을 이용하기 쉽다는 것이다. 몇 가지 Concurrency pattern은 다음과 같다.

* `Worker pool` pattern - 요청이 있을 때마다 Worker(Goroutine)이 생겨나며 그 개수에 제한이 없는 것이 아니라 일정한 pool size만큼만 goroutine을 생성하고, 요청만 channel을 통해 부분 부분 전달해주는 패턴
* `Pipeline` pattern - A 단계에서 작업 처리를 모두 완료한 뒤에 그 결과를 통째로 리턴하고, 그것을 다음 단계인 B 단계가 입력으로서 이용하는 것이 아니라 A 단계에서는 실시간으로 작업 처리를 전달할 `channel`을 리턴하고 B 단계도 실시간으로 데이터를 전달받을 수 있는 channel을 입력으로 사용하는 패턴. channel을 이용해 A와 B 단계 사이의 pipline이 생성된다고 볼 수 있다.
* `Fan-in Fan-out` pattern - Fan-in이란 여러 Worker(goroutine)로부터 1개의 channel에 데이터가 들어오는 것을 말하고, Fan-out은 여러 Worker(goroutine)으로부터 1개의 channel의 데이터가 빠져나가는 것을 말한다. 주로 어떤 worker가 1개의 channel 속 데이터를 가져가게 될 지는 idle한 worker 중 random하게 정해진다고 볼 수 있다.

Fan-in Fan-out pattern 속에 pipeline pattern이 속할 수도 있는 등 각각의 pattern은 서로 완전히 독립된  별개의 패턴이 아닌 듯 하다.

bumblebee는 이미지 처리 시에 Fan-in Fan-out pattern을 사용하고 있고, 사실상 Fan-in Fan-out을 위해선 각 step간의 데이터 전달을 위한 pipeline pattern, N개의 worker들이 한 channel에서 데이터를 가져가기 위한 worker pool pattern 등이 적용되어야했다.

## 사용된 Concurrency pattern에 대하여

1. 간단히 말하자면 Fan-in Fan-out pattern을 사용 중. 
2. 사용자의 요청에 응답하는 HttpHandling gorutine들(개수가 정해지지 않음)이 1개의 이미지 리사이징 작업 channel에 메시지를 전달
3. 이미지 리사이징 워커들인 M개의 goroutine들이 해당 channel에서 메시지를 꺼내어 리사이징 작업
4. 리사이징 작업이 완료되면 업로드 작업 channel에 메시지를 전달한다.
5. 업로드 워커인 L개의 goroutine들이 해당 channel에서 메시지를 꺼내어 실제 업로드 작업을 하는 goroutine을 실행시킨다.

이렇게 concurrency pattern을 이용할 때의 장점에 대해 알아보자.



**장점 1. 전체 작업이 늘어지지 않고, 앞의 작업은 우선적으로 처리될 수 있다.**

CPU Bound한 작업을 동시적으로 수행하려는 경우 CPU의 한계 상 작업들이 골고루 수행되어야하므로 thread나 goroutine이 많을 수록 전체 작업들이 늘어진다. 예를 들어 30개의 요청이 비슷한 시기에 들어온다면 30개의 이미지 리사이징 goroutine이 생성되고, 30개가 골고루 모두 작업이 끝날 때쯤 우루루 작업이 완료된다.

반면 수행이미지 리사이징 작업을 예를 들어 6개의 goroutine이 담당하도록 Worker pool을 이용한다면 30개의 요청이 왔다고해서 30개의 고루틴이 생성된 뒤 전체 30개의 작업이 늘어지는 것이 아니라 6개 작업 단위로 바로 바로 처리된다고 볼 수 있다. Throughput면에서는 별 차이가 없지만 개별 작업 면에서는 latency가 크게 감소하고, Memory 효율면에서도 전체 작업에 대한 메모리를 계속 점유할 필요 없이 완료된 작업에 대한 메모리는 먼저 해제될 수 있으니 우수하다.



**장점 2. 메시지 브로커 서비스 이용 없이 메시지 기능을 이용할 수 있다.**

Image 서버 하나를 하면서 App내의 메시지 전달을 위해 Kafka나 RabbitMQ와 같은 서비스를 이용하는 것은 효율적이지도 않고 성가실 것이다.

일반적인 프로그래밍적 방식으로는 Resize 함수 호출 후 Upload 함수를 호출하면 간단하긴하다. 하지만 어딘가 Resize 호출 => Upload 호출을 관리하는 로직도 필요할 것이고 그곳에서 Resize나 Upload에 대한 에러 처리도 담당을 해야할 것이다. Resize 함수 내에 Upload 함수를 포함시킨다면 Resize 함수 내에서 Upload에 대한 에러 처리를 담당해야할 수도 있다.

하지만 **pipeline 혹은 fan-in fan-out 패턴을 이용하면 메시징 시스템을 이용할 때와 마찬가지로 한 작업은 메시지 전달만 시켜놓고 그 뒤로 그 메시지가 어떻게 사용되는지는 알 필요 없다.** 따라서 에러 처리도 이후에 작업을 처리하는 각자가 알아서하면 된다. 예를 들어 Resize 함수는 리사이징 작업 완료 후 Upload task라는 메시지만 upload task channel에 전송하면 되고 Upload 과정에서 어떤 에러가 발생하든 그것은 Resize 함수의 관심 밖이다. Upload는 누가 자기를 호출하는지 알 필요 없이 그냥 upload task channel에서 메시지(task)만 꺼내어 작업하면 된다.

단점 - 메시지 브로커 서비스와 달리 앱이 죽으면 메모리에 채널을 통해 저장 중이던 메시지가 소실된다.

## 리사이징 Worker pool Benchmark

과연 정말 리사이징 작업에 Worker pool pattern을 적용할 때가 제한 없이 goroutine을 생성해서 이용할 때보다 효율적일까 벤치마크를 통해 알아보고자한다.

사용된 Machine: AWS EC2 t2.micro

작업: 2MB의 Image에 대한 128x128로의 리사이징 요청 30개를 처리.

```bash
goos: linux
goarch: amd64
pkg: github.com/khu-dev/bumblebee
BenchmarkTransformer_Start/30_task_worker_pool_1         	       5	 426898179 ns/op
BenchmarkTransformer_Start/30_task_worker_pool_10        	       5	 424531051 ns/op
BenchmarkTransformer_Start/30_task_worker_pool_19        	       5	 423642287 ns/op
BenchmarkTransformer_Start/30_task_worker_pool_28        	       5	 424124379 ns/op
BenchmarkTransformer_Start/30_task_unlimited_concurrency 	       5	 425058364 ns/op
PASS
ok  	github.com/khu-dev/bumblebee	21.476s
```

예상했던 대로 throughput은 큰 차이가 없었다. 하지만 먼저 들어온 작업은 먼저 처리될 수 있다는 점이 장점일 것 같다.

워커(goroutine)의 수를 N이라고 했을 때

`N = 1`: 먼저 들어온 작업이 무조건 먼저 처리된다. 만약 앞선 작업이 오랜 시간을 소모한다면 뒤의 작업들도 모두 지연된다.

`N=무한대`: 먼저 들어온 작업과 나중에 들어온 작업이 모두 늘어지면서 한꺼번에 처리된다.



따라서 먼저 들어온 작업이 먼저 처리될 수 있으면서 오랜 작업 시간을 소모하는 작업이 앞에 위치해도 뒤의 작업들이 완전히 지연되기보다는 어느 정도 먼저 처리될 수도 있도록 하기 위해 적당히 3개~5개 정도의 goroutine을 이용하는 것이 어떨까싶다.

> 원래는 논리적 프로세서 개수보다 워커 수가 적으면 성능이 아주 안좋아지지만, 리사이징에 사용하는 라이브러 자체가 하나의 작업도 논리적 프로세서 만큼으로 쪼개어 병렬적으로 처리하기 때문에 내가 정의한 goroutine이 논리적 프로세서 개수보다 적더라도 속도가 떨어지지 않고있다.

### 아쉬운 점

(2021.01.24) 이미지 리사이징 라이브러리의 코드를 까보니 이미 논리적 프로세서 개수만큼의 goroutine으로 작업하게 최적화가 되어있었고, 이미지 리사이징 작업이 생각보다 오래 걸리는 작업이 아니었다. 우리 쿠뮤가 애초에 이미지 업로드 요청이 많을 서비스는 아님에도 기술적 야망으로 인해 이미지 프로세싱 작업을 마이크로서비스로 분리해 동시성 패턴을 적용시켜보았다. 하지만 동시성을 이렇게 조절해볼까 저렇게 조절해볼까 했던 것에 비해 결과에는 큰 차이가 없었던 것 같아 조금 아쉽다.

[당근마켓의 이미지 처리 관련 글](https://medium.com/daangn/lambda-edge%EB%A1%9C-%EA%B5%AC%ED%98%84%ED%95%98%EB%8A%94-on-the-fly-%EC%9D%B4%EB%AF%B8%EC%A7%80-%EB%A6%AC%EC%82%AC%EC%9D%B4%EC%A7%95-f4e5052d49f3)을 보면 2019.01 기준 하루 50만장의 이미지 업로드가 이뤄진다고했다. [당근마켓 이용자 변화](https://brunch.co.kr/@trendlite/12)를 보면 이때 기준으로 이용자가 약 2021년엔 4배 증가했을 것으로 예상된다. 그럼 약 200만장의 이미지가 업로드 된다고 가정, 자고 일하는 시간 제외 그 200만장의 이미지는 하루 24시간이 아니라 실질적으로 6시간정도 동안 업로드 된다고 가정하면 1분당 약 5500장 정도의 이미지가 업로드 되는 셈이라고 추정해볼 수 있겠다. 이런 대규모 서비스에서는 이미지 프로세싱 서버에서 작업을 Go의 채널과 고루틴을 이용한 동시성 패턴을 적용해 작업 큐처럼 수행하면 어느정도 이점이 있을 수 있을 것 같다. 추후에 쿠뮤에도 뭔가 고화질 이미지를 많이 업로드할 만한 서비스가 추가되면 좋겠다.

(2021.01.28) 생각보다 고화질(약 5MB 이상) 이미지의 경우 리사이징 작업이 메모리와 CPU를 많이 잡아먹는 것으로 보인다. Python이나 Node.js에서 작업할 때는 자원 소모가 어떻게 될 지 궁금하다. 현재로서는 Go로 수행 중인 이 작업이 자원 소모 측면에서 효율적인지 알 수 있도록 해주는 비교군이 없다.

## Test code

* JUnit에서 아이디어를 얻어 BeforeEach, AfterEach 등을 정의함으로써 **각 test case들 간의 의존성을 없앰**.
* AfterEach에서 test 수행 후 업로드한 리사이징 된 이미지등을 지움으로써 깔끔하게 이용.
* 개발을 하면서 결과 확인을 위해 매번 특정 함수를 실행하기 위한 커맨드나 단축키를 이용할 필요 없이 file watcher에서 test code를 실행하도록 설정해놓으면 되기 때문에 간편.

## 아키텍처

이미지를 리사이징 후 업로드해서 사용자들이 접근할 수 있게 해야하므로 정적 리소스들을 제공할 클라우드 인프라가 필요하다.

AWS의 `S3` + `CloudFront` + `Route53`을 이용했다. 이미지 업로드에 대한 API의 도메인 네임은 다른 API와 동일하지만
업로드한 이미지는 https://api.xxx.xxx가 아닌 https://storage.xxx.xxx를 루트로 사용한다.

* **S3 public bucket**
* **CloudFront**
  * Origin - S3 public bucket
  * Alternative CNAME - 
* **Route53**
  * GoDaddy에서 구매한 도메인은 GCP의 CloudDNS의 NS에 연결되어있음.
  * Route53에서 drive.khumu.me Hosted Zone 생성
  * GCP의 CloudDNS에서 Route53의 drive.khumu.me Hosted Zone NS를 레코드로 추가
  
## 간단한 부하 테스트

```bash
$ for ((i=1;i<=100;i++)); 
do curl -F 'image=@test_data_wallpaper.jpg' http://localhost:9001/api/images
done
```

로컬에서 서버 실행 후 위의 커맨드를 통해 작업을 요청하고 CPU, Memory 사용률을 관찰해본다.

## 이미지 처리 시 Exif Metadata의 Orientation 정보

* Orientation 값에 따른 회전 정보 참고 - https://feel5ny.github.io/2018/08/06/JS_13/
* Exif 데이터 해석 참고 - https://github.com/dsoprea/go-exif
  * jpeg의 Exif 데이터 추출 참고 - https://pkg.go.dev/github.com/dsoprea/go-jpeg-image-structure
  * png의 Exif 데이터 추출 참고 - https://pkg.go.dev/github.com/dsoprea/go-png-image-structure