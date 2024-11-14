package http

import (
	"actors/platform/actor/head"
	"actors/platform/system"
	"actors/platform/utils"
	"actors/proto"
	"actors/proto/dag"
	"actors/proto/deploy"
	"bytes"
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	pb "google.golang.org/protobuf/proto"
)

type Response struct {
	Status int    `json:"status"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Actor struct {
	app  *gin.Engine
	port int
}

func (a *Actor) Receive(ctx actor.Context) {}

func (a *Actor) App() *gin.Engine {
	return a.app
}

func (a *Actor) OnInit(ctx actor.Context) {
	a.app = gin.Default(func(e *gin.Engine) {
		e.POST("/deploy", func(c *gin.Context) {
			resp := &Response{}
			defer c.JSON(resp.Status, resp)

			data := new(bytes.Buffer)
			if _, err := data.ReadFrom(c.Request.Body); err != nil {
				resp.Status = 400
				resp.Error = fmt.Sprintf("failed reading from request body: %v", err)
				return
			}
			cmd := &deploy.Command{}
			if err := pb.Unmarshal(data.Bytes(), cmd); err != nil {
				resp.Status = 400
				resp.Error = fmt.Sprintf("failed decoding command: %v", err)
				return
			}

			fut := ctx.RequestFuture(head.ActorRef().DeployManager(), cmd, system.ExecutionTimeout)
			result, err := fut.Result()
			if err != nil {
				resp.Status = 500
				resp.Error = fmt.Sprintf("failed executing command: %v", err)
				return
			}

			switch result := result.(type) {
			case *proto.Error:
				resp.Status = 500
				resp.Error = result.Message
			case *proto.Ack:
				resp.Status = 200
			default:
				resp.Status = 500
				resp.Error = fmt.Sprintf("unexpected result type: %T", result)
			}
		})

		e.POST("/dag", func(c *gin.Context) {
			resp := &Response{}
			defer c.JSON(resp.Status, resp)

			data := new(bytes.Buffer)
			if _, err := data.ReadFrom(c.Request.Body); err != nil {
				resp.Status = 400
				resp.Error = fmt.Sprintf("failed reading from request body: %v", err)
				return
			}

			cmd := &dag.Command{}
			if err := pb.Unmarshal(data.Bytes(), cmd); err != nil {
				resp.Status = 400
				resp.Error = fmt.Sprintf("failed decoding command: %v", err)
				return
			}

			fut := utils.NewFuture[any](system.ExecutionTimeout)
			props := actor.PropsFromFunc(func(ctx actor.Context) {
				switch result := ctx.Message().(type) {
				case *actor.Started:
					return
				case *proto.Error:
					resp.Status = 500
					resp.Error = result.Message
				case *proto.Ack:
					resp.Status = 200
				case *dag.ExecutionResult:
					resp.Status = 200
					resp.Result = result
				default:
					resp.Status = 500
					resp.Error = fmt.Sprintf("unexpected result type: %T", result)
				}

				fut.Resolve(nil)
			})
			cmd.ReplyTo = ctx.Spawn(props)
			ctx.Send(head.ActorRef().DAGManager(), cmd)

			_, err := fut.Result()
			if err != nil {
				resp.Status = 500
				resp.Error = fmt.Sprintf("failed executing command: %v", err)
				return
			}
		})
	})

	go a.app.Run(fmt.Sprintf(":%d", a.port))
}

func NewActor(port int) *actor.Props {
	self := &Actor{port: port}
	return actor.PropsFromProducer(func() actor.Actor {
		return self
	}, actor.WithOnInit(self.OnInit))
}

func Start(port int) {
	system.ActorSystem().Root.Spawn(NewActor(port))
}
