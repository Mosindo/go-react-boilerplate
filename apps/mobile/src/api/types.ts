export type User = {
  id: string;
  email: string;
  createdAt: string;
};

export type Post = {
  id: string;
  authorUserId: string;
  title: string;
  body: string;
  createdAt: string;
  updatedAt: string;
};

export type Comment = {
  id: string;
  postId: string;
  authorUserId: string;
  content: string;
  createdAt: string;
  updatedAt: string;
};

export type Notification = {
  id: string;
  userId: string;
  type: string;
  title: string;
  body: string;
  isRead: boolean;
  createdAt: string;
  readAt?: string;
};

// Matches GET /chats response items (direct conversation summaries).
export type Conversation = {
  user: User;
  lastMessage?: {
    content: string;
    createdAt: string;
  };
};

export type Message = {
  id: string;
  senderUserId: string;
  recipientUserId: string;
  content: string;
  createdAt: string;
};
