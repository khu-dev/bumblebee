# bumblebee - 이미지 처리 서버

`bumblebee`는 쿠뮤의 이미지 처리 작업을 담당하는 마이크로서비스이다. Go의 Worker pool pattern이나 pipeline pattern, Fan-in Fan-out pattern과 같은 여러 Concurrency pattern(동시성 패턴)들을 적용시켜보고자했으며 기존의 단순한 함수 호출 형태와는 다른 구조를 갖고있다.

## Concurrency pattern in Go

Go 언어의 큰 특징 중 하나는 다양한 Concurrency pattern을 이용하기 쉽다는 것이다. 몇 가지 Concurrency pattern은 다음과 같다.

* Worker pool pattern - 요청이 있을 때마다 Worker(Goroutine)이 생겨나며 그 개수에 제한이 없는 것이 아니라 일정한 pool size만큼만 goroutine을 생성하고, 요청만 channel을 통해 부분 부분 전달해주는 패턴
* Pipeline pattern - A 단계에서 작업 처리를 모두 완료한 뒤에 그 결과를 통째로 리턴하고, 그것을 다음 단계인 B 단계가 입력으로서 이용하는 것이 아니라 A 단계에서는 실시간으로 작업 처리를 전달할 `channel`을 리턴하고 B 단계도 실시간으로 데이터를 전달받을 수 있는 channel을 입력으로 사용하는 패턴. channel을 이용해 A와 B 단계 사이의 pipline이 생성된다고 볼 수 있다.
* Fan-in Fan-out pattern - Fan-in이란 여러 Worker(goroutine)로부터 1개의 channel에 데이터가 들어오는 것을 말하고, Fan-out은 여러 Worker(goroutine)으로부터 1개의 channel의 데이터가 빠져나가는 것을 말한다. 주로 어떤 worker가 1개의 channel 속 데이터를 가져가게 될 지는 idle한 worker 중 random하게 정해진다고 볼 수 있다.

Fan-in Fan-out pattern 속에 pipeline pattern이 속할 수도 있는 등 각각의 pattern은 서로 완전히 독립된  별개의 패턴이 아닌 듯 하다.

bumblebee는 이미지 처리 시에 Fan-in Fan-out pattern을 사용하고 있고, 사실상 Fan-in Fan-out을 위해선 각 step간의 데이터 전달을 위한 pipeline pattern, N개의 worker들이 한 channel에서 데이터를 가져가기 위한 worker pool pattern 등이 적용되어야했다.

## 사용된 Concurrency pattern에 대하여

1. 간단히 말하자면 Fan-in Fan-out pattern을 사용 중. 사용자의 요청에 응답하는 N개의 goroutine이 1개의 이미지 리사이징 작업 channel에 메시지를 전달하고 이미지 리사이징 워커들인 M개의 goroutine들이 해당 channel에서 메시지를 꺼내어 작업한다.
2. 리사이징 작업이 완료되면 업로드 작업 channel에 메시지를 전달한다.
3. 업로드 워커인 L개의 goroutine들이 해당 channel에서 메시지를 꺼내어 작업한다.

이렇게 concurrency pattern을 이용할 때의 장점에 대해 알아보자.



**전체 작업이 늘어지지 않고, 앞의 작업은 우선적으로 처리될 수 있다.**

CPU Bound한 작업을 동시적으로 수행하려는 경우 CPU의 한계 상 작업들이 골고루 수행되어야하므로 thread나 goroutine이 많을 수록 전체 작업들이 늘어진다. 예를 들어 30개의 요청이 비슷한 시기에 들어온다면 30개의 이미지 리사이징 goroutine이 생성되고, 30개가 골고루 모두 작업이 끝날 때쯤 우루루 작업이 완료된다.

반면 수행이미지 리사이징 작업을 예를 들어 6개의 goroutine이 담당하도록 Worker pool을 이용한다면 30개의 요청이 왔다고해서 30개의 고루틴이 생성된 뒤 전체 30개의 작업이 늘어지는 것이 아니라 6개 작업 단위로 바로 바로 처리된다고 볼 수 있다. Throughput면에서는 별 차이가 없지만 개별 작업 면에서는 latency가 크게 감소하고, Memory 효율면에서도 전체 작업에 대한 메모리를 계속 점유할 필요 없이 완료된 작업에 대한 메모리는 먼저 해제될 수 있으니 우수하다.



**메시지 브로커 서비스 이용 없이 메시지 기능을 이용할 수 있다.**

Image 서버 하나를 하면서 App내의 메시지 전달을 위해 Kafka나 RabbitMQ와 같은 서비스를 이용하는 것은 효율적이지도 않고 성가실 것이다.

일반적인 프로그래밍적 방식으로는 Resize 함수 호출 후 Upload 함수를 호출하면 간단하긴하다. 하지만 어딘가 Resize 호출 => Upload 호출을 관리하는 로직도 필요할 것이고 그곳에서 Resize나 Upload에 대한 에러 처리도 담당을 해야할 것이다. Resize 함수 내에 Upload 함수를 포함시킨다면 Resize 함수 내에서 Upload에 대한 에러 처리를 담당해야할 수도 있다.

하지만 pipeline 혹은 fan-in fan-out 패턴을 이용하면 메시징 시스템을 이용할 때와 마찬가지로 한 작업은 메시지 전달만 시켜놓고 그 메시지가 어떻게 사용되는지, 올바르게 처리 됐는지는 알 필요가 없다. 따라서 에러 처리도 각자가 알아서하면 된다. Resize 함수는 Upload task라는 메시지만 upload task channel에 전송하면 되고 Upload는 누가 자기를 호출하는지 알 필요 없이 그냥 upload task channel에서 메시지(task)만 꺼내어 작업하면 된다.

## 리사이징 Worker pool Benchmark

과연 정말 리사이징 작업에 Worker pool pattern을 적용할 때가 제한 없이 goroutine을 생성해서 이용할 때보다 효율적일까 벤치마크를 통해 알아보고자한다.

(다음에)

그리고 Machine 혹은 container 환경에 따라 Worker pool size를 어떻게 했을 때 더 효율적인지 알아보고자한다.

(다음에)



## TDD

