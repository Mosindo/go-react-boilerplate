import type { NavigatorScreenParams } from "@react-navigation/native";
import type { UserSummary } from "../api/types";

export type ProfileStackParamList = {
  ProfileHome: undefined;
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  BlockedUsers: undefined;
  DeleteAccount: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Profile: NavigatorScreenParams<ProfileStackParamList> | undefined;
};

export type RootStackParamList = {
  Auth: undefined;
  Onboarding: undefined;
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { conversationId: string; user: UserSummary };
  UserProfile: { userId: string; conversationId?: string };
};
