import grpc
import service_pb2
import service_pb2_grpc

def run():
    # Create a channel to the gRPC server
    with grpc.insecure_channel('localhost:50051') as channel:
        stub = service_pb2_grpc.StreamServiceStub(channel)
        
        # Create a StreamRequest
        request = service_pb2.StreamRequest(query="Hello, server!")
        
        # Open a stream with the server
        response_stream = stub.StreamData(request)

        # Iterate through the responses from the server
        try:
            for response in response_stream:
                print("Received:", response.data)
        except grpc.RpcError as e:
            print(f"RPC failed: {e.code()} - {e.details()}")
        except Exception as e:
            print(f"Unexpected error: {e}")

if __name__ == "__main__":
    run()