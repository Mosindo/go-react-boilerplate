import type { NavigatorScreenParams } from "@react-navigation/native";
import type { SwipeAction } from "../api/types";

export type TabParamList = {
  Discover: { decision?: { userId: string; action: SwipeAction; nonce: number } } | undefined;
  Matches: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
  Onboarding: undefined;
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Conversation: { userId: string; firstName?: string; photoUrl?: string | null };
  ProfileDetail: { userId: string; mode: "discovery" | "match" | "view" };
  EditProfile: undefined;
  EditPhotos: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
};

declare global {
  namespace ReactNavigation {
    // eslint-disable-next-line @typescript-eslint/no-empty-object-type
    interface RootParamList extends RootStackParamList {}
  }
}
