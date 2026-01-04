package worker

type Pool struct {
	jobs chan func()
}

func New(size int) *Pool {
	pool := &Pool{jobs: make(chan func(), 64)}
	for i := 0; i < size; i++ {
		go func() {
			for job := range pool.jobs {
				job()
			}
		}()
	}
	return pool
}

func (p *Pool) Submit(job func()) {
	p.jobs <- job
}

var Default = New(2)

func Submit(job func()) {
	Default.Submit(job)
}
