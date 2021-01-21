package main

var(
    TransformerWorkers []*Transformer
)
func main(){
    InitConfig()
    StartTransformerWorkers()
    keepServerRunning := make(chan struct{})
    keepServerRunning <- struct{}{}
}

func StartTransformerWorkers(){
    TransformerWorkers := make([]*Transformer, Config.NumOfTransformerWorkers)
    for i := 0; i < Config.NumOfTransformerWorkers; i++{
        TransformerWorkers[i] = &Transformer{}
        transformer := TransformerWorkers[i]
        go transformer.Start()
    }
}