import {
  useNavigation,
  useRoute,
  type NavigationProp,
  type ParamListBase,
  type RouteProp
} from "@react-navigation/native";
import type { UserSummary } from "./types";

export type ChatRouteParams = { conversationId: string; user: UserSummary };
export type UserProfileRouteParams = { userId: string; conversationId?: string };

export function useAppNavigation(): NavigationProp<ParamListBase> {
  return useNavigation<NavigationProp<ParamListBase>>();
}

export function useChatParams(): ChatRouteParams {
  return useRoute<RouteProp<{ Chat: ChatRouteParams }, "Chat">>().params;
}

export function useUserProfileParams(): UserProfileRouteParams {
  return useRoute<RouteProp<{ UserProfile: UserProfileRouteParams }, "UserProfile">>().params;
}

export function openChat(
  navigation: NavigationProp<ParamListBase>,
  conversationId: string,
  user: UserSummary
): void {
  navigation.navigate("Chat", { conversationId, user });
}

export function openUserProfile(
  navigation: NavigationProp<ParamListBase>,
  params: UserProfileRouteParams
): void {
  navigation.navigate("UserProfile", params);
}
