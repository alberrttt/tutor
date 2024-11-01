<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import Tail from '../tail.svelte';
	import { cubicOut } from 'svelte/easing';
	import { SvelteSet } from 'svelte/reactivity';
	import katex from 'katex';
	import {
		type Message,
		type UserMessage,
		type AssistantMessage,
		type ProblemMessage,
		is_user,
		is_assistant,
		is_problem
	} from '$lib/util';
	import { writable } from 'svelte/store';

	let previousIndex = $state(0);
	let apiToken = $state('');
	let messages = $state<Message[]>(window.__messages || []);
	$effect(() => {
		window.__messages = messages;
	});
	let chat_box_text = $state('');
	let waitingForResponse = $state(false);
	let isModalOpen = $state(false);
	let chatName = $state('');
	let chatMap = $state<{ [key: string]: Message[] }>({});

	// New state to track shown solutions
	let shownSolutions = $state(new SvelteSet<number>());

	interface ChatResponse {
		type: string;
		content: string;
		problem_solutions: ProblemSolution[];
		elapsed_time_ms: number;
	}
	interface ProblemSolution {
		question: string;
		answer: string;
	}

	async function sendMessage() {
		const user_message = chat_box_text;
		chat_box_text = '';
		messages = [...messages, { text: user_message, role: 'user' }];
		messages.push({ role: 'assistant', text: '', duration: 0 });

		try {
			waitingForResponse = true;
			const response = await fetch(
				`http://localhost:8080/chat?message=${encodeURIComponent(user_message)}`
			);
			const data: ChatResponse = await response.json();
			messages[messages.length - 1] = {
				role: 'assistant',
				text: data.content,
				duration: data.elapsed_time_ms
			};
			console.log(messages);
			for (const problem of data.problem_solutions) {
				messages.push({
					role: 'problem',
					question: problem.question,
					solution: problem.answer,
					msg: messages.length - 1
				});
			}
		} catch (error) {
			console.error('Error:', error);
		}
		waitingForResponse = false;
		window.__messages = messages;
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey && !waitingForResponse) {
			event.preventDefault();
			sendMessage();
		}
	}

	function katexRender(node: HTMLElement, math: string) {
		try {
			katex.render(math, node, {
				throwOnError: false
			});
		} catch (error) {
			node.textContent = math;
			console.error('KaTeX render error:', error);
		}
		return {};
	}

	onMount(() => {
		chatMap = JSON.parse(window.localStorage.getItem('chatMap') || '{}');
	});

	function new_chat() {
		messages = [];
		fetch(`http://localhost:8080/new`);
	}

	// Updated toggleSolution function
	function toggleSolution(index: number) {
		if (shownSolutions.has(index)) {
			shownSolutions.delete(index);
		} else {
			shownSolutions.add(index);
		}
	}
	type ParsedText = Array<
		| { latex: string }
		| { text: string }
		| { bold: string }
		| { italic: string }
		| { code: string }
		| { link: string }
		| { heading: string }
	>;
	const delimiters = [
		// LaTeX delimiters
		{ type: 'latex', start: '$$', end: '$$' },
		{ type: 'latex', start: '\\(', end: '\\)' },
		{ type: 'latex', start: '\\\\begin{equation}', end: '\\\\end{equation}' },
		{ type: 'latex', start: '\\\\\\[', end: '\\\\]' },
		// Bold delimiters
		{ type: 'bold', start: '**', end: '**' },
		{ type: 'bold', start: '__', end: '__' },
		// Italic delimiters
		{ type: 'italic', start: '*', end: '*' },
		{ type: 'italic', start: '_', end: '_' },
		// Code block delimiters
		{ type: 'code', start: '```', end: '```' },
		{ type: 'code', start: '`', end: '`' },
		// Heading delimiters
		{ type: 'heading', start: '### ', end: '\n' },
		{ type: 'heading', start: '## ', end: '\n' },
		{ type: 'heading', start: '# ', end: '\n' }
	];
	function parseText(text: string): ParsedText {
		const result: ParsedText = [];
		let currentIndex = 0;

		while (currentIndex < text.length) {
			let nextDelimiter = null;
			let delimiterIndex = Infinity;

			for (const delimiter of delimiters) {
				const index = text.indexOf(delimiter.start, currentIndex);
				if (index !== -1 && index < delimiterIndex) {
					delimiterIndex = index;
					nextDelimiter = delimiter;
				}
			}

			if (delimiterIndex === Infinity) {
				// No more delimiters found, add remaining text
				if (currentIndex < text.length) {
					result.push({ text: text.slice(currentIndex) });
				}
				break;
			} else if (delimiterIndex > currentIndex) {
				// Add text before the delimiter
				result.push({ text: text.slice(currentIndex, delimiterIndex) });
			}

			if (nextDelimiter) {
				const start = delimiterIndex + nextDelimiter.start.length;
				let end;
				if (nextDelimiter.type === 'heading') {
					end = text.indexOf(nextDelimiter.end, start);
					if (end === -1) end = text.length;
				} else {
					end = text.indexOf(nextDelimiter.end, start);
				}

				if (end === -1) {
					// No matching end delimiter found, treat the rest as plain text
					result.push({ text: text.slice(delimiterIndex) });
					break;
				}

				const content = text.slice(start, end);
				switch (nextDelimiter.type) {
					case 'bold':
						result.push({ bold: content });
						break;
					case 'italic':
						result.push({ italic: content });
						break;
					case 'code':
						result.push({ code: content });
						break;
					case 'heading':
						result.push({ heading: content.trim() });
						break;
					case 'latex':
						result.push({ latex: content });
						break;
					default:
						result.push({ text: content });
				}

				currentIndex = end + nextDelimiter.end.length;
			}
		}

		return result;
	}
	const filtered_messages = $derived(
		messages.filter((m) => {
			if (is_problem(m)) {
				return !!m.question && !!m.solution;
			}
			return true;
		})
	);
</script>

<div class="layout">
	<!-- <div class="sidebar chat-history">
		{#if Object.keys(chatMap).length > 0}
			<ul class="chat-list">
				{#each Object.keys(chatMap) as chatName}
					<li class="chat-item flex justify-between items-center">
						<button class="chat-name cursor-pointer hover:underline" type="button">
							{chatName}
						</button>
						<button
							class="delete-button text-red-500 hover:text-red-700"
							aria-label={`Delete chat ${chatName}`}
						>
							&#10005;
						</button>
					</li>
				{/each}
			</ul>
		{:else}{/if}
	</div> -->
	<div class="chat-container flex flex-col justify-between">
		<span class="fit-content flex flex-row justify-end">
			<div class="header-bar gap-2">
				<!-- <button class="button" aria-label="save" onclick={openModal}>
				<svg
					class="stroke-zinc-50 w-6 h-6"
					data-slot="icon"
					aria-hidden="true"
					fill="none"
					stroke-width="2px"
					viewBox="0 0 24 24"
					xmlns="http://www.w3.org/2000/svg"
				>
					<path
						d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z"
						stroke-linecap="round"
						stroke-linejoin="round"
					></path>
				</svg>
			</button> -->
				<button class="button" aria-label="newchat" onclick={new_chat}>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="24"
						height="24"
						viewBox="0 0 24 24"
						fill="none"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						class="feather feather-plus stroke-zinc-50"
					>
						<line x1="12" y1="5" x2="12" y2="19"></line>
						<line x1="5" y1="12" x2="19" y2="12"></line>
					</svg>
				</button>
			</div></span
		>
		<span class="flex flex-col justify-between overflow-hidden">
			<div class="messages-container overflow-x-hidden">
				{#each filtered_messages as message, i}
					<span
						class={`flex flex-row ${is_user(message) ? 'self-end' : 'self-start'} relative break-words`}
					>
						{#if is_problem(message)}
							<div
								class="p-4 font-nunito bg-yellow-100 rounded-md border-l-4 border-amber-500 flex flex-col"
							>
								<div class="inline whitespace-pre-wrap">
									{#each parseText(message.question) as token}
										{#if 'latex' in token}
											<span class="inline-block" use:katexRender={token.latex}> </span>
										{:else if 'text' in token}
											{token.text}
										{:else if 'bold' in token}
											<span class="font-bold">{token.bold}</span>
										{:else if 'italic' in token}
											<span class="italic">{token.italic}</span>
										{:else if 'code' in token}
											<span class="bg-gray-100 p-1 rounded">{token.code}</span>
										{:else if 'link' in token}
											<a href={token.link} class="text-blue-500 hover:underline">{token.link}</a>
										{:else if 'heading' in token}
											<span class="text-lg font-bold">{token.heading}</span>
										{/if}
									{/each}
									{#if shownSolutions.has(i)}
										{#each parseText(message.solution) as token}
											{#if 'latex' in token}
												<span class="inline-block" use:katexRender={token.latex}> </span>
											{:else}
												<p class="mt-2 text-green-700 italic">
													{#if 'text' in token}
														{token.text}
													{:else if 'bold' in token}
														<span class="font-bold">{token.bold}</span>
													{:else if 'italic' in token}
														<span class="italic">{token.italic}</span>
													{:else if 'code' in token}
														<span class="bg-gray-100 p-1 rounded">{token.code}</span>
													{:else if 'link' in token}
														<a href={token.link} class="text-blue-500 hover:underline"
															>{token.link}</a
														>
													{:else if 'heading' in token}
														<span class="text-lg font-bold">{token.heading}</span>
													{/if}
												</p>
											{/if}
										{/each}
									{/if}
								</div>
								<button
									class="mt-2 max-w-fit px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
									onclick={() => toggleSolution(i)}
								>
									{shownSolutions.has(i) ? 'Hide' : 'Show'} Solution
								</button>
							</div>
						{:else}
							<span
								class={`chat-bubble rounded-lg p-3 break-words ${
									is_user(message) ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-800'
								} flex flex-row relative`}
							>
								{#if waitingForResponse && i === messages.length - 1}
									<div class="dot-container p-3 flex flex-row items-center">
										<div class="dot"></div>
										<div class="dot"></div>
										<div class="dot"></div>
									</div>
								{:else}
									<div class="inline message-text font-nunito whitespace-pre-wrap">
										{#if is_user(message)}
											{message.text}
										{:else if is_assistant(message)}
											<span class="flex flex-col">
												<span class="inline message-text font-nunito whitespace-pre-wrap">
													{#each parseText(message.text) as token}
														{#if 'latex' in token}
															<span class="inline-block" use:katexRender={token.latex}> </span>
														{:else if 'text' in token}
															{token.text}
														{:else if 'text' in token}
															{token.text}
														{:else if 'bold' in token}
															<span class="font-bold">{token.bold}</span>
														{:else if 'italic' in token}
															<span class="italic">{token.italic}</span>
														{:else if 'code' in token}
															<span class="bg-gray-100 p-1 rounded">{token.code}</span>
														{:else if 'link' in token}
															<a href={token.link} class="text-blue-500 hover:underline"
																>{token.link}</a
															>
														{:else if 'heading' in token}
															<span class="text-lg font-bold">{token.heading}</span>
														{/if}
													{/each}
												</span>
												<span class="text-xs text-gray-500">
													Took {message.duration}ms
												</span>
											</span>
										{/if}
									</div>
								{/if}
							</span>
							<Tail
								class={`self-end absolute ${
									is_user(message)
										? 'right-[-9px] fill-blue-600'
										: 'left-[-9px] scale-x-[-1] fill-gray-200'
								}`}
							/>
						{/if}
					</span>
				{/each}
			</div>
			<div class="input-container mt-2">
				<textarea
					class="flex-grow border-none bg-transparent resize-none
					 font-nunito focus:outline-none"
					rows="3"
					bind:value={chat_box_text}
					placeholder="Type your message here..."
					onkeydown={handleKeyDown}
				></textarea>
				<button class="send-button" onclick={sendMessage} aria-label="Send message">
					{#if waitingForResponse}
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="24"
							height="24"
							viewBox="0 0 24 24"
							fill="none"
							stroke="white"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
						</svg>
					{:else}
						<svg
							stroke="white"
							xmlns="http://www.w3.org/2000/svg"
							width="24"
							height="24"
							viewBox="0 0 24 24"
							fill="none"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							class="feather feather-corner-down-left"
						>
							<polyline points="9 10 4 15 9 20"></polyline>
							<path d="M20 4v7a4 4 0 0 1-4 4H4"></path>
						</svg>
					{/if}
				</button>
			</div>
		</span>
	</div>
</div>

<style>
	.dot-container {
		display: flex;
		gap: 4px;
	}

	.dot {
		width: 4px;
		height: 4px;
		background-color: #4848488c;
		border-radius: 50%;
		animation: bounce 1.5s infinite;
	}

	.dot:nth-child(2) {
		animation-delay: 0.3s;
	}

	.dot:nth-child(3) {
		animation-delay: 0.6s;
	}

	@keyframes bounce {
		0%,
		20%,
		50%,
		80%,
		100% {
			transform: translateY(0);
		}
		40% {
			transform: translateY(-10px);
		}
		60% {
			transform: translateY(-5px);
		}
	}
	.layout {
		display: flex;
		height: 100vh;
		background-color: #f0f0f0;
	}

	.sidebar {
		width: 250px;
		padding: 20px;
		background-color: black;
		color: white;
		box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
	}

	.sidebar-title {
		font-size: 1.5rem;
		margin-bottom: 20px;
	}

	.divider {
		width: 1px;
		background-color: #ccc;
		height: 100%;
	}

	.chat-container {
		flex: 1;
		background-color: #fff;

		display: flex;
		flex-direction: column;
		@apply my-12 mx-20 p-4;
		box-shadow: 0 4px 10px rgba(0, 0, 0, 0.2);
		border-radius: 15px;
	}

	.messages-container {
		flex-grow: 1;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 8px;
		@apply rounded-md px-4 mx-1;
	}

	.message-text {
		flex-grow: 1;
	}

	.input-container {
		display: flex;
		align-items: center;

		background-color: #f9f9f9;

		@apply rounded-xl p-4;
	}

	.send-button {
		width: 48px;
		height: 48px;
		margin-left: 8px;
		border: none;
		border-radius: 50%;
		background-color: #222;
		display: flex;
		justify-content: center;
		align-items: center;
		cursor: pointer;
		transition: background-color 0.3s;
	}

	.send-button:hover {
		background-color: #111;
	}
	.header-bar {
		@apply flex flex-row justify-between shadow-md max-w-fit bg-neutral-900 mb-2 p-4;
		border-top-right-radius: 1rem;
		border-bottom-right-radius: 1rem;
		border-top-left-radius: 1.5rem;
		border-bottom-left-radius: 1.5rem;
		position: relative;
		overflow: hidden;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
</style>
