export interface UserMessage {
    role: 'user';
    text: string;
}

export interface AssistantMessage {
    role: 'assistant';
    text: string;
    duration: number;
}

export interface ProblemMessage {
    role: 'problem';
    solution: string;
    question: string;
    // the index of the assistant message that this problem is a response to
    msg:number
}

export type Message = UserMessage | AssistantMessage | ProblemMessage;
declare global {
    interface Window {
        __messages: Message[];
        katex: any;
    }
}

export function is_user(message: Message): message is UserMessage {
    return message.role === 'user';
}

export function is_assistant(message: Message): message is AssistantMessage {
    return message.role === 'assistant';
}

export function is_problem(message: Message): message is ProblemMessage {
    return message.role === 'problem';
}
