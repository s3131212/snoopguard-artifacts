package treekem

import (
	pb "chatbot-poc-go/pkg/protos/services"
)

func nodeEqual(n1, n2 *Node) bool {
	return string(n1.Public) == string(n2.Public)
}

func groupEqual(g1, g2 *TreeKEMState) bool {
	if g1.Size() != g2.Size() {
		return false
	}

	for i := 0; i < g1.Size(); i++ {
		lhn := g1.Nodes()[i]
		rhn := g2.Nodes()[i]
		if lhn == nil || rhn == nil {
			continue
		}

		if !nodeEqual(lhn, rhn) {
			return false
		}
	}

	return true
}

/*
PbTreeKEMUserAddConvert converts the TreeKEMUserAdd in protobuf to UserAdd object.
*/
func PbTreeKEMUserAddConvert(pbTreeKEMUserAdd *pb.TreeKEMUserAdd) UserAdd {
	return UserAdd{
		Size:        int(pbTreeKEMUserAdd.GetSize()),
		Ciphertexts: PbECKEMCipherTextMapSliceConvert(pbTreeKEMUserAdd.GetCiphertexts()),
		Nodes:       PbTreeKEMNodeMapConvert(pbTreeKEMUserAdd.GetNodes()),
	}
}

/*
PbTreeKEMGroupInitKeyConvert converts the TreeKEMGroupInitKey in protobuf to GroupInitKey object.
*/
func PbTreeKEMGroupInitKeyConvert(pbTreeKEMGroupInitKey *pb.TreeKEMGroupInitKey) GroupInitKey {
	return GroupInitKey{
		Size:     int(pbTreeKEMGroupInitKey.GetSize()),
		Frontier: PbTreeKEMNodeMapConvert(pbTreeKEMGroupInitKey.GetFrontier()),
	}
}

/*
PbTreeKEMUserUpdateConvert converts the TreeKEMUserUpdate in protobuf to UserUpdate object.
*/
func PbTreeKEMUserUpdateConvert(pbTreeKEMUserUpdate *pb.TreeKEMUserUpdate) UserUpdate {
	return UserUpdate{
		From:        int(pbTreeKEMUserUpdate.GetFrom()),
		Ciphertexts: PbECKEMCipherTextMapSliceConvert(pbTreeKEMUserUpdate.GetCiphertexts()),
		Nodes:       PbTreeKEMNodeMapConvert(pbTreeKEMUserUpdate.GetNodes()),
	}
}

/*
PbECKEMCipherTextConvert converts the ECKEMCipherText in protobuf to ECKEMCipherText object.
*/
func PbECKEMCipherTextConvert(pbECKEMCipherText *pb.ECKEMCipherText) ECKEMCipherText {
	return ECKEMCipherText{
		Public:     pbECKEMCipherText.GetPublic(),
		IV:         pbECKEMCipherText.GetIV(),
		CipherText: pbECKEMCipherText.GetCipherText(),
	}
}

/*
PbECKEMCipherTextMapConvert converts the ECKEMCipherTextMap in protobuf to a map of ECKEMCipherText object.
*/
func PbECKEMCipherTextMapConvert(pbECKEMCipherTextMap *pb.ECKEMCipherTextMap) map[int]ECKEMCipherText {
	cipherTexts := make(map[int]ECKEMCipherText)
	for index, pbECKEMCipherText := range pbECKEMCipherTextMap.GetCiphertexts() {
		cipherTexts[int(index)] = PbECKEMCipherTextConvert(pbECKEMCipherText)
	}
	return cipherTexts
}

/*
PbECKEMCipherTextMapSliceConvert converts a ECKEMCipherTextMap in protobuf to a slice of maps of ECKEMCipherText object.
*/
func PbECKEMCipherTextMapSliceConvert(pbECKEMCipherTextMapSlice []*pb.ECKEMCipherTextMap) []map[int]ECKEMCipherText {
	cipherTexts := make([]map[int]ECKEMCipherText, len(pbECKEMCipherTextMapSlice))
	for index, pbECKEMCipherTextMap := range pbECKEMCipherTextMapSlice {
		cipherTexts[index] = PbECKEMCipherTextMapConvert(pbECKEMCipherTextMap)
	}
	return cipherTexts
}

/*
PbECKEMCipherTextStringMapConvert converts the ECKEMCipherTextStringMap in protobuf to a map of ECKEMCipherText object.
*/
func PbECKEMCipherTextStringMapConvert(pbECKEMCipherTextStringMap *pb.ECKEMCipherTextStringMap) map[string]ECKEMCipherText {
	cipherTexts := make(map[string]ECKEMCipherText)
	for index, pbECKEMCipherText := range pbECKEMCipherTextStringMap.GetCiphertexts() {
		cipherTexts[index] = PbECKEMCipherTextConvert(pbECKEMCipherText)
	}
	return cipherTexts
}

/*
PbTreeKEMNodeConvert converts the TreeKEMNode in protobuf to Node object.
*/
func PbTreeKEMNodeConvert(pbTreeKEMNode *pb.TreeKEMNode) *Node {
	return &Node{
		Secret:      pbTreeKEMNode.GetSecret(),
		Public:      pbTreeKEMNode.GetPublic(),
		Private:     pbTreeKEMNode.GetPrivate(),
		SignPublic:  pbTreeKEMNode.GetSignPublic(),
		SignPrivate: pbTreeKEMNode.GetSignPrivate(),
	}
}

/*
PbTreeKEMNodeMapConvert converts a map of TreeKEMNode in protobuf to a map of Node object.
*/
func PbTreeKEMNodeMapConvert(pbTreeKEMNodes map[uint32]*pb.TreeKEMNode) map[int]*Node {
	nodes := make(map[int]*Node)
	for index, pbTreeKEMNode := range pbTreeKEMNodes {
		nodes[int(index)] = PbTreeKEMNodeConvert(pbTreeKEMNode)
	}
	return nodes
}

/*
TreeKEMUserAddPbConvert converts the UserAdd object to TreeKEMUserAdd in protobuf.
*/
func TreeKEMUserAddPbConvert(userAdd UserAdd) *pb.TreeKEMUserAdd {
	return &pb.TreeKEMUserAdd{
		Size:        uint32(userAdd.Size),
		Ciphertexts: ECKEMCipherTextMapSlicePbConvert(userAdd.Ciphertexts),
		Nodes:       TreeKEMNodeMapPbConvert(userAdd.Nodes),
	}
}

/*
TreeKEMGroupInitKeyPbConvert converts a GroupInitKey object to TreeKEMGroupInitKey in protobuf.
*/
func TreeKEMGroupInitKeyPbConvert(groupInitKey GroupInitKey) *pb.TreeKEMGroupInitKey {
	return &pb.TreeKEMGroupInitKey{
		Size:     uint32(groupInitKey.Size),
		Frontier: TreeKEMNodeMapPbConvert(groupInitKey.Frontier),
	}
}

/*
ECKEMCipherTextMapSlicePbConvert converts a slice of map of ECKEMCipherText object to a slice of ECKEMCipherTextMap in protobuf.
*/
func ECKEMCipherTextMapSlicePbConvert(cipherTexts []map[int]ECKEMCipherText) []*pb.ECKEMCipherTextMap {
	pbCipherTexts := make([]*pb.ECKEMCipherTextMap, len(cipherTexts))
	for index, cipherText := range cipherTexts {
		pbCipherTexts[index] = ECKEMCipherTextMapPbConvert(cipherText)
	}
	return pbCipherTexts
}

/*
ECKEMCipherTextMapPbConvert converts a map of ECKEMCipherText object to ECKEMCipherTextMap in protobuf.
*/
func ECKEMCipherTextMapPbConvert(cipherText map[int]ECKEMCipherText) *pb.ECKEMCipherTextMap {
	pbCipherText := make(map[uint32]*pb.ECKEMCipherText)
	for index, c := range cipherText {
		pbCipherText[uint32(index)] = ECKEMCipherTextPbConvert(&c)
	}
	return &pb.ECKEMCipherTextMap{
		Ciphertexts: pbCipherText,
	}
}

/*
ECKEMCipherTextStringMapPbConvert converts a map of ECKEMCipherText object to ECKEMCipherTextMap in protobuf.
*/
func ECKEMCipherTextStringMapPbConvert(cipherText map[string]ECKEMCipherText) *pb.ECKEMCipherTextStringMap {
	pbCipherText := make(map[string]*pb.ECKEMCipherText)
	for index, c := range cipherText {
		pbCipherText[index] = ECKEMCipherTextPbConvert(&c)
	}
	return &pb.ECKEMCipherTextStringMap{
		Ciphertexts: pbCipherText,
	}
}

/*
ECKEMCipherTextPbConvert converts a ECKEMCipherText object to ECKEMCipherText in protobuf.
*/
func ECKEMCipherTextPbConvert(cipherText *ECKEMCipherText) *pb.ECKEMCipherText {
	if cipherText == nil {
		return nil
	}
	return &pb.ECKEMCipherText{
		Public:     cipherText.Public,
		IV:         cipherText.IV,
		CipherText: cipherText.CipherText,
	}
}

/*
TreeKEMNodeMapPbConvert converts a map of Node object to TreeKEMNode in protobuf.
*/
func TreeKEMNodeMapPbConvert(nodes map[int]*Node) map[uint32]*pb.TreeKEMNode {
	pbNodes := make(map[uint32]*pb.TreeKEMNode)
	for index, node := range nodes {
		pbNodes[uint32(index)] = TreeKEMNodePbConvert(node)
	}
	return pbNodes
}

/*
TreeKEMNodePbConvert converts a Node object to TreeKEMNode in protobuf.
*/
func TreeKEMNodePbConvert(node *Node) *pb.TreeKEMNode {
	return &pb.TreeKEMNode{
		Secret:      node.Secret,
		Public:      node.Public,
		Private:     node.Private,
		SignPublic:  node.SignPublic,
		SignPrivate: node.SignPrivate,
	}
}

/*
TreeKEMUserUpdatePbConvert converts a UserUpdate object to TreeKEMUserUpdate in protobuf.
*/
func TreeKEMUserUpdatePbConvert(userUpdate *UserUpdate) *pb.TreeKEMUserUpdate {
	if userUpdate == nil {
		return nil
	}
	return &pb.TreeKEMUserUpdate{
		From:        uint32(userUpdate.From),
		Ciphertexts: ECKEMCipherTextMapSlicePbConvert(userUpdate.Ciphertexts),
		Nodes:       TreeKEMNodeMapPbConvert(userUpdate.Nodes),
	}
}
