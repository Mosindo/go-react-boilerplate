import type { NavigatorScreenParams } from "@react-navigation/native";
import type { PublicProfile } from "../api/types";

export type TabParamList = {
  Home: undefined;
  Matches: undefined;
  Chat: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type ProfileSource = "discover" | "likes" | "match";

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Thread: { matchId: string; profile: PublicProfile };
  ProfileDetail: { userId: string; source?: ProfileSource; matchId?: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Blocked: undefined;
};
