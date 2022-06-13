
package main

import (
	"fmt"
	"net"
	"net/rpc"
	"log"
)

// Responder para obter respostas como mensagens OK das chamadas RPC.
type Reply struct{
	Data string
}

// Estrutura do algoritmo Core Bully. Contém funções registradas para RPC.
//
type BullyAlgorithm struct{
	my_id int
	coordinator_id int
	ids_ip map[int]string
}

// Se um site já invocou a eleição, ele não precisa iniciar as eleições novamente
var no_election_invoked = true

// Esta é a função de eleição que é invocada quando um ID de host menor solicita uma eleição para este host
func (bully *BullyAlgorithm) Election(invoker_id int, reply *Reply) error{
	fmt.Println("Log: Recebendo a eleição de", invoker_id)
	if invoker_id < bully.my_id{
		fmt.Println("Log: Enviando OK para", invoker_id)
		reply.Data = "OK"				// envia mensagem de OK para o site pequeno
		if no_election_invoked{
			no_election_invoked = false
			go invokeElection()			// invoca a eleição para seus anfitriões superiores
		}
	}
	return nil
}

var superiorNodeAvailable = false				// Alternado quando qualquer host superior envia mensagem OK

// Esta função invoca a eleição para seus hosts superiores. Ele envia seu Id como parâmetro ao chamar o RPC
func invokeElection(){
	for id,ip := range bully.ids_ip{
		reply := Reply{""}
		if id > bully.my_id{
			fmt.Println("Log: Enviando eleição para", id)
			client, error := rpc.Dial("tcp",ip)
			if error != nil{
				fmt.Println("Log:", id, "não está disponível.")
				continue
			}
			err := client.Call("BullyAlgorithm.Election", bully.my_id, &reply)
			if err != nil{
				fmt.Println(err)
				fmt.Println("Log: Erro ao chamar a função", id, "election")
				continue
			}
			if reply.Data == "OK"{				// Significa que o host superior existe
				fmt.Println("Log: Recebido OK de", id)
				superiorNodeAvailable = true
			}
		}
	}
	if !superiorNodeAvailable{					// se nenhum site superior estiver ativo, o host pode se tornar o coordenador
		makeYourselfCoordinator()
	}
	superiorNodeAvailable = false
	no_election_invoked = true					// redefinir a eleição invocada
}

// Esta função é chamada pelo novo Coordenador para atualizar as informações do coordenador dos outros hosts
func (bully *BullyAlgorithm) NewCoordinator(new_id int, reply *Reply) error{
	bully.coordinator_id = new_id 
	fmt.Println("Log:", bully.coordinator_id, "agora é o novo coordenador")
	return nil
}

func (bully *BullyAlgorithm) HandleCommunication(req_id int, reply *Reply) error{
	fmt.Println("Log: Recebendo comunicação de", req_id)
	reply.Data = "OK"
	return nil
}

func communicateToCoordinator(){
	coord_id := bully.coordinator_id
	coord_ip := bully.ids_ip[coord_id]
	fmt.Println("Log: Coordenador de comunicação", coord_id)
	my_id := bully.my_id
	reply := Reply{""}
	client, err := rpc.Dial("tcp", coord_ip)
	if err != nil{
		fmt.Println("Log: Coordenador",coord_id, "comunicação falhou!")
		fmt.Println("Log: Invocando eleições")
		invokeElection()
		return
	}
	err = client.Call("BullyAlgorithm.HandleCommunication", my_id, &reply)
	if err != nil || reply.Data != "OK"{
		fmt.Println("Log: Coordenador de comunicação", coord_id, "Falhou!")
		fmt.Println("Log: Invocando eleições")
		invokeElection()
		return
	}
	fmt.Println("Log: Comunicação recebida do coordenador", coord_id)
}

// Esta função é chamada quando o host decide que é o coordenador.
// ele transmite a mensagem para todos os outros hosts e atualiza as informações do líder, incluindo seu próprio host.
func makeYourselfCoordinator(){
	reply := Reply{""}
	for id, ip := range bully.ids_ip{
		client, error := rpc.Dial("tcp", ip)
		if error != nil{
			fmt.Println("Log:", id, "erro de comunicação")
			continue
		}
		client.Call("BullyAlgorithm.NewCoordinator", bully.my_id, &reply)
	}
}

// Objeto central do algoritmo bully inicializado com todos os endereços IP de todos os outros sites na rede
var bully = BullyAlgorithm{
	my_id: 		1,
	coordinator_id: 5,
	ids_ip: 	map[int]string{	1:"127.0.0.1:3000", 2:"127.0.0.1:3001", 3:"127.0.0.1:3002", 4:"127.0.0.1:3003", 5:"127.0.0.1:3004"}}


func main(){
	my_id := 0
	fmt.Printf("Insira o código de ID [1-5]: ")			// inicialize o ID do host em tempo de execução
	fmt.Scanf("%d", &my_id)
	bully.my_id = my_id
	my_ip := bully.ids_ip[bully.my_id]
	address, err := net.ResolveTCPAddr("tcp", my_ip) 
	if err != nil{
		log.Fatal(err)
	}
	inbound, err := net.ListenTCP("tcp", address)
	if err != nil{
		log.Fatal(err)
	}
	rpc.Register(&bully)
	fmt.Println("servidor está sendo executado com endereço IP e número da porta:", address)
	go rpc.Accept(inbound) // Aceitando conexões de outros hosts.

	reply := ""
	fmt.Printf("Este nó está se recuperando de uma falha?(y/n): ")	// Recupera de uma falha.
	fmt.Scanf("%s", &reply)
	if reply == "y"{
		fmt.Println("Log: Invocando eleições")
		invokeElection()
	}

	random := ""
	for{
		fmt.Printf("Pressione enter para %d comunicar-se com o coordenador.\n", bully.my_id)
		fmt.Scanf("%s", &random)
		communicateToCoordinator()
		fmt.Println("")
	}
	fmt.Scanf("%s", &random)
}
