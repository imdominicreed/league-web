/**
 * Page Objects Index
 *
 * Re-exports all page object classes for easy importing.
 *
 * Usage:
 *   import { HomePage, LoginPage, DraftRoomPage } from '../page-objects';
 */

export { BasePage } from './base.page';
export { HomePage } from './home.page';
export { LoginPage } from './login.page';
export { RegisterPage } from './register.page';
export { CreateLobbyPage } from './create-lobby.page';
export { LobbyRoomPage } from './lobby-room.page';
export { DraftRoomPage, waitForAnyTurn } from './draft-room.page';
