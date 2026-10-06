import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Likes: undefined;
  Messages: undefined;
  Me: undefined;
};

export type ProfileContext = "discover" | "likes" | "match";

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  ProfileDetail: { userId: string; context: ProfileContext; matchId?: string };
  Chat: { matchId: string; peerId: string; peerName: string };
  Notifications: undefined;
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
  Report: { userId: string; name: string };
};

export type AuthStackParamList = {
  Auth: undefined;
  ForgotPassword: undefined;
};
