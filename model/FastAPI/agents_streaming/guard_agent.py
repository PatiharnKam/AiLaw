from dotenv import load_dotenv
import os, json
from copy import deepcopy
from .utils import get_chatbot_response, evaluate_safety_response, get_guard_classification, chunked_safety_scan
from .prompts import GUARD_SYSTEM_PROMPTS, SAFETY_TAXONOMY
# from openai import OpenAI
from openai import AsyncOpenAI
from pinecone import Pinecone
# from groq import Groq
from groq import AsyncGroq
load_dotenv()

class GuardAgent():
    def __init__(self):
        self.client = AsyncOpenAI(
            api_key = os.getenv("TYPHOON_API"),
            base_url = os.getenv("RUNPOD_CHATBOT_URL")
        )
        self.client_groq = AsyncGroq(
            api_key=os.getenv("groq_api")
        )
        self.model_name = os.getenv("MODEL_NAME")
        self.model_guard = os.getenv("model_guard")
        self.model_prompt_guard = os.getenv("model_prompt_guard")
        self.token = os.getenv("RUNPOD_API_KEY")
        self.embedding_url = os.getenv("RUNPOD_EMBEDDING_URL")
        self.pc = Pinecone(api_key = os.getenv("PINECONE_API_KEY"))
        self.system_prompt = GUARD_SYSTEM_PROMPTS
        self.SAFETY_TAXONOMY = SAFETY_TAXONOMY

        
    def postprocess(self, output):
        output = json.loads(output)
        dict_output = {
            "role" : "assistant",
            "content" : output['message'],
            "memory" : {
                "agent" : "gurad_agent",
                "guard_decision" : output['decision']
            }
        }
        return dict_output
    
    
    async def get_response(self, messages):
        """
        Function to get response from LLM model (3 Layers)
        """

        messages = deepcopy(messages)
        input_messages = [
            {"role": "system", "content": self.system_prompt}
        ] + messages
        prompt = input_messages[1]['content']
        
        # Layer 1
        print("Guard Layer 1 : Start .....")
        layer1_output = await get_guard_classification(client=self.client_groq,
                                                    model_name=self.model_guard,
                                                    text=prompt)
        is_safe, layer1_message = evaluate_safety_response(guard_output=layer1_output,
                                                           SAFETY_TAXONOMY=self.SAFETY_TAXONOMY)
        if is_safe==False:
            output = self.postprocess(layer1_message)
            return output
        print(layer1_message)
        
        # Layer 2
        print("Guard Layer 2 : Start .....")
        is_safe, layer2_message = await chunked_safety_scan(client=self.client_groq,
                                                      model_name=self.model_prompt_guard,
                                                      long_text=prompt)
        if is_safe==False:
            output = self.postprocess(layer2_message)
            return output
        print(layer2_message)
        
        # Layer 3
        print("Guard Layer 3 : Start .....")
        layer3_output = await get_chatbot_response(client=self.client, 
                                             model_name=self.model_name,
                                             messages=input_messages)
        output = self.postprocess(layer3_output)
        print("Guard Layer 3 : End.....")
        return output
    
