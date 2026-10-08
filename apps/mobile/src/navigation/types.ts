import type { NavigatorScreenParams } from "@react-navigation/native";

export type ChatUser = { id: string; firstName: string; photoPath?: string };

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Messages: undefined;
  Me: undefined;
};

export type AuthStackParamList = {
  Auth: undefined;
  Forgot: { email?: string } | undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  ProfileDetail: { userId: string; matchId?: string; conversationId?: string; canSwipe?: boolean };
  Chat: { conversationId: string; user: ChatUser; matchId?: string };
  EditProfile: undefined;
  Preferences: undefined;
  Settings: undefined;
  Notifications: undefined;
  Blocked: undefined;
  ChangePassword: undefined;
  Report: { userId: string; firstName: string };
};
