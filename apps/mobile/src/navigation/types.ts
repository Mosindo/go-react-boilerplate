import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { matchId: string; userId: string; name: string };
  UserProfile: { userId: string; matchId?: string };
  EditProfile: undefined;
  Settings: undefined;
  Blocked: undefined;
  Report: { userId: string; name: string };
};
