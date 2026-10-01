import type { NavigatorScreenParams } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { PublicProfile } from "../api/types";

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  ProfileDetail: { userId: string; profile?: PublicProfile; canSwipe?: boolean };
  Conversation: { conversationId: string; userId?: string; user?: PublicProfile };
  EditProfile: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type RootScreenProps<T extends keyof RootStackParamList> = NativeStackScreenProps<RootStackParamList, T>;
