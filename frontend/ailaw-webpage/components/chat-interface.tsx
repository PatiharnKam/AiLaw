// "use client"

// import type React from "react"

// import { useState } from "react"
// import { Button } from "@/components/ui/button"
// import { Input } from "@/components/ui/input"
// import { ScrollArea } from "@/components/ui/scroll-area"
// import { Avatar, AvatarFallback } from "@/components/ui/avatar"
// import { Card } from "@/components/ui/card"
// import { Send, Plus, MessageSquare, Settings, Moon, Sun, Menu, X } from "lucide-react"
// import { cn } from "@/lib/utils"

// interface Message {
//   id: string
//   content: string
//   role: "user" | "assistant"
//   timestamp: Date
// }

// interface Conversation {
//   id: string
//   title: string
//   messages: Message[]
//   lastMessage: Date
// }

// export function ChatInterface() {
//   const [messages, setMessages] = useState<Message[]>([
//     {
//       id: "1",
//       content: "Hello! I'm your AI assistant. How can I help you today?",
//       role: "assistant",
//       timestamp: new Date(),
//     },
//   ])
//   const [inputValue, setInputValue] = useState("")
//   const [conversations, setConversations] = useState<Conversation[]>([
//     {
//       id: "1",
//       title: "New Chat",
//       messages: [],
//       lastMessage: new Date(),
//     },
//   ])
//   const [activeConversation, setActiveConversation] = useState("1")
//   const [isDarkMode, setIsDarkMode] = useState(false)
//   const [sidebarOpen, setSidebarOpen] = useState(false)

//   const handleSendMessage = () => {
//     if (!inputValue.trim()) return

//     const newMessage: Message = {
//       id: Date.now().toString(),
//       content: inputValue,
//       role: "user",
//       timestamp: new Date(),
//     }

//     setMessages((prev) => [...prev, newMessage])
//     setInputValue("")

//     // Simulate AI response
//     setTimeout(() => {
//       const aiResponse: Message = {
//         id: (Date.now() + 1).toString(),
//         content:
//           "Thank you for your message! This is a simulated response from the AI assistant. In a real implementation, this would connect to your AI service.",
//         role: "assistant",
//         timestamp: new Date(),
//       }
//       setMessages((prev) => [...prev, aiResponse])
//     }, 1000)
//   }

//   const handleKeyPress = (e: React.KeyboardEvent) => {
//     if (e.key === "Enter" && !e.shiftKey) {
//       e.preventDefault()
//       handleSendMessage()
//     }
//   }

//   const startNewChat = () => {
//     const newConversation: Conversation = {
//       id: Date.now().toString(),
//       title: "New Chat",
//       messages: [],
//       lastMessage: new Date(),
//     }
//     setConversations((prev) => [newConversation, ...prev])
//     setActiveConversation(newConversation.id)
//     setMessages([
//       {
//         id: "1",
//         content: "Hello! I'm your AI assistant. How can I help you today?",
//         role: "assistant",
//         timestamp: new Date(),
//       },
//     ])
//     setSidebarOpen(false)
//   }

//   const toggleTheme = () => {
//     setIsDarkMode(!isDarkMode)
//     document.documentElement.classList.toggle("dark")
//   }

//   return (
//     <div className={cn("flex h-screen bg-background", isDarkMode && "dark")}>
//       {/* Sidebar */}
//       <div
//         className={cn(
//           "fixed inset-y-0 left-0 z-50 w-64 bg-sidebar border-r border-sidebar-border transform transition-transform duration-200 ease-in-out lg:translate-x-0 lg:static lg:inset-0",
//           sidebarOpen ? "translate-x-0" : "-translate-x-full",
//         )}
//       >
//         <div className="flex flex-col h-full">
//           {/* Sidebar Header */}
//           <div className="flex items-center justify-between p-4 border-b border-sidebar-border">
//             <h1 className="text-lg font-semibold text-sidebar-foreground">ChatBot</h1>
//             <Button variant="ghost" size="sm" onClick={() => setSidebarOpen(false)} className="lg:hidden">
//               <X className="h-4 w-4" />
//             </Button>
//           </div>

//           {/* New Chat Button */}
//           <div className="p-4">
//             <Button
//               onClick={startNewChat}
//               className="w-full justify-start gap-2 bg-sidebar-primary text-sidebar-primary-foreground hover:bg-sidebar-primary/90"
//             >
//               <Plus className="h-4 w-4" />
//               New Chat
//             </Button>
//           </div>

//           {/* Conversations List */}
//           <ScrollArea className="flex-1 px-4">
//             <div className="space-y-2">
//               {conversations.map((conversation) => (
//                 <Button
//                   key={conversation.id}
//                   variant={activeConversation === conversation.id ? "secondary" : "ghost"}
//                   className="w-full justify-start gap-2 text-left h-auto py-3 px-3"
//                   onClick={() => {
//                     setActiveConversation(conversation.id)
//                     setSidebarOpen(false)
//                   }}
//                 >
//                   <MessageSquare className="h-4 w-4 flex-shrink-0" />
//                   <span className="truncate text-sm">{conversation.title}</span>
//                 </Button>
//               ))}
//             </div>
//           </ScrollArea>

//           {/* Sidebar Footer */}
//           <div className="p-4 border-t border-sidebar-border space-y-2">
//             <Button variant="ghost" className="w-full justify-start gap-2" onClick={toggleTheme}>
//               {isDarkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
//               {isDarkMode ? "Light Mode" : "Dark Mode"}
//             </Button>
//             <Button variant="ghost" className="w-full justify-start gap-2">
//               <Settings className="h-4 w-4" />
//               Settings
//             </Button>
//           </div>
//         </div>
//       </div>

//       {/* Main Chat Area */}
//       <div className="flex-1 flex flex-col min-w-0">
//         {/* Chat Header */}
//         <div className="flex items-center justify-between p-4 border-b border-border bg-card">
//           <div className="flex items-center gap-3">
//             <Button variant="ghost" size="sm" onClick={() => setSidebarOpen(true)} className="lg:hidden">
//               <Menu className="h-4 w-4" />
//             </Button>
//             <h2 className="font-semibold text-card-foreground">AI Assistant</h2>
//           </div>
//         </div>

//         {/* Messages Area */}
//         <ScrollArea className="flex-1 p-4">
//           <div className="max-w-3xl mx-auto space-y-6">
//             {messages.map((message) => (
//               <div
//                 key={message.id}
//                 className={cn("flex gap-3", message.role === "user" ? "justify-end" : "justify-start")}
//               >
//                 {message.role === "assistant" && (
//                   <Avatar className="h-8 w-8 flex-shrink-0">
//                     <AvatarFallback className="bg-primary text-primary-foreground text-sm">AI</AvatarFallback>
//                   </Avatar>
//                 )}

//                 <Card
//                   className={cn(
//                     "max-w-[80%] p-4",
//                     message.role === "user"
//                       ? "bg-primary text-primary-foreground ml-12"
//                       : "bg-card text-card-foreground",
//                   )}
//                 >
//                   <p className="text-sm leading-relaxed whitespace-pre-wrap">{message.content}</p>
//                   <p
//                     className={cn(
//                       "text-xs mt-2 opacity-70",
//                       message.role === "user" ? "text-primary-foreground" : "text-muted-foreground",
//                     )}
//                   >
//                     {message.timestamp.toLocaleTimeString([], {
//                       hour: "2-digit",
//                       minute: "2-digit",
//                     })}
//                   </p>
//                 </Card>

//                 {message.role === "user" && (
//                   <Avatar className="h-8 w-8 flex-shrink-0">
//                     <AvatarFallback className="bg-secondary text-secondary-foreground text-sm">You</AvatarFallback>
//                   </Avatar>
//                 )}
//               </div>
//             ))}
//           </div>
//         </ScrollArea>

//         {/* Input Area */}
//         <div className="p-4 border-t border-border bg-card">
//           <div className="max-w-3xl mx-auto">
//             <div className="flex gap-2">
//               <Input
//                 value={inputValue}
//                 onChange={(e) => setInputValue(e.target.value)}
//                 onKeyPress={handleKeyPress}
//                 placeholder="Type your message here..."
//                 className="flex-1 bg-input border-border focus:ring-ring"
//               />
//               <Button
//                 onClick={handleSendMessage}
//                 disabled={!inputValue.trim()}
//                 className="bg-primary text-primary-foreground hover:bg-primary/90"
//               >
//                 <Send className="h-4 w-4" />
//               </Button>
//             </div>
//             <p className="text-xs text-muted-foreground mt-2 text-center">
//               Press Enter to send, Shift + Enter for new line
//             </p>
//           </div>
//         </div>
//       </div>

//       {/* Overlay for mobile sidebar */}
//       {sidebarOpen && (
//         <div className="fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden" onClick={() => setSidebarOpen(false)} />
//       )}
//     </div>
//   )
// }
