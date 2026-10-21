import type { Match } from "../api/types";

export type AuthStackParamList = {
  Auth: undefined;
  ForgotPassword: undefined;
};

export type RootStackParamList = {
  Tabs: undefined;
  ProfileDetail: { userId: string; canSwipe?: boolean };
  Chat: { conversationId: string; userId: string; name: string };
  Notifications: undefined;
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  BlockedUsers: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
  Report: { userId: string; name: string };
  Match: { match: Match };
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Profile: undefined;
};
