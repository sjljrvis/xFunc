import grpc
import coder_pb2
import coder_pb2_grpc
import sys

def run():
    # Create a channel to the gRPC server
    with grpc.insecure_channel('localhost:50051') as channel:
        stub = coder_pb2_grpc.CoderServiceStub(channel)
        
        # Create a StreamRequest
        system_prompt = """
        You are a python coder
        When given a task to do, you MUST follow below rules :-
        1. You MUST reply in markdown with properly labelled code blocks which should indicate type of programming language for-eg : python.
        2. You MUST provide installation steps where pip install should be a bash command.
        3. You MUST also mention filename in the code block appropriately for eg "# filename : <appropriate_file_name>.py" . Remember to mention filename in codeblock.
        4. You MUST provide steps to execute and it should be a bash command, Remeber this else there is a penalty.
        5. You MUST always include complete python program which can be executed as it is, without any modifications.
        6. You MUST wait for user to respond weather code ran successfully with exit-code 0.
        7. If user responds with exit-code with expected output only then reply with word "TERMINATE" only at the end to indicate to user that he can stop now.
        8. Do not respond with code block if expected out is returned with exit-code 0.
        """
        request = coder_pb2.CodeRequest(
            systemPrompt= system_prompt,
            userPrompt="write code to print multiplication 2^5 * 7^4",
            workingDirectory="/Users/sejal/Personal/codexec/coding",
            dockerImage = "code.buildpack.python",
            LLMModel= 'gpt-3.5-turbo',
            maxRetry= 3
        )
        
        # Open a stream with the server
        response_stream = stub.ExecuteCode(request)

        # Iterate through the responses from the server
        try:
            for response in response_stream:
                # sys.stdout.write(response.data)
                print(response.data, end='')
        except grpc.RpcError as e:
            print(f"RPC failed: {e.code()} - {e.details()}")
        except Exception as e:
            print(f"Unexpected error: {e}")

        # print(f"Code execution result: {response.result}")

if __name__ == "__main__":
    run()


#  code -> execute -> exit 0 -> llm-response - terminate